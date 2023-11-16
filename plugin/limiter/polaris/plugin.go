//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 THL A29 Limited, a Tencent company.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

// Package polaris is a rate limiting plugin for tRPC-Gateway based on Polaris.
package polaris

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/config"
	"github.com/tidwall/gjson"
	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/convert"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	gwplugin "trpc.group/trpc-go/trpc-gateway/common/plugin"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
)

const (
	// PluginName is the name of the plugin.
	PluginName = "polaris_limiter"
	// ErrLimit indicates that the request has been rate limited.
	ErrLimit = 10003
)

func init() {
	plugin.Register(PluginName, &PluginFactory{})
}

// PluginFactory is the plugin factory
type PluginFactory struct {
	// Default timeout for rate limiting requests
	Timeout int `yaml:"timeout"`
	// Default number of retries for rate limiting requests
	MaxRetries int `yaml:"max_retries"`
}

// Type implements the Type method of plugin.Factory.
func (p *PluginFactory) Type() string {
	return gwplugin.DefaultType
}

// Setup implements the Setup method of plugin.Factory, initializes and registers the interceptor.
func (p *PluginFactory) Setup(_ string, decoder plugin.Decoder) error {
	if err := decoder.Decode(p); err != nil {
		return gerrs.Wrap(err, "decode polaris limiter config err")
	}

	var err error
	limiter, err := New(WithTimeout(p.Timeout), WithMaxRetries(p.MaxRetries))
	if err != nil {
		return gerrs.Wrap(err, "failed to create polaris limitAPI")
	}

	filter.Register(PluginName, limiter.InterceptServer, nil)
	if err := gerrs.Register(ErrLimit, fasthttp.StatusTooManyRequests); err != nil {
		return gerrs.Wrap(err, "register rate limit code err")
	}
	return nil
}

// Limiter is the Polaris rate limiter.
type Limiter struct {
	// Timeout for individual quota queries, in milliseconds
	Timeout int `yaml:"timeout"`
	// Number of retries
	MaxRetries int `yaml:"max_retries"`
	// Global rate limiting object
	API api.LimitAPI
}

// New creates a new Limiter.
var New = func(opts ...Opt) (*Limiter, error) {
	l := &Limiter{}
	for _, o := range opts {
		o(l)
	}
	cfg := config.NewDefaultConfigurationWithDomain()
	limitAPI, err := api.NewLimitAPIByConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create limitAPI from default configuration, err: %w", err)
	}
	l.API = limitAPI

	return l, nil
}

// Opt allows customization of the Limiter.
type Opt func(*Limiter)

// WithTimeout sets the timeout for rate limiting requests.
func WithTimeout(timeout int) Opt {
	return func(l *Limiter) {
		l.Timeout = timeout
	}
}

// WithMaxRetries sets the maximum number of retries for failed rate limiting requests.
func WithMaxRetries(maxRetries int) Opt {
	return func(l *Limiter) {
		l.MaxRetries = maxRetries
	}
}

// Options for rate limiting
type Options struct {
	// Business fields
	Labels []string `yaml:"labels" json:"labels"`
	// Timeout for quota query, in milliseconds
	Timeout int `yaml:"timeout" json:"timeout"`
	// Number of retries
	MaxRetries int `yaml:"max_retries" json:"max_retries"`
	// Polaris rate limiting service name, defaults to the gateway project's rate limiting configuration, can be overridden
	Service string `yaml:"service" json:"service"`
	// Polaris rate limiting namespace, defaults to the gateway instance's namespace
	Namespace string `yaml:"namespace" json:"namespace"`
	// Response body after rate limiting
	LimitedRspBody string `yaml:"limited_rsp_body" json:"limited_rsp_body"`
	// Retrieve rate limiting parameters from JSON request body
	ParseJSONBody bool `yaml:"parse_json_body" json:"parse_json_body"`
}

// CheckConfig validates the plugin configuration and returns a parsed configuration object. Used in the ServerFilter
// method.
func (p *PluginFactory) CheckConfig(name string, decoder plugin.Decoder) error {
	// Convert the map structure to the target struct. Called during gateway configuration initialization.
	options := &Options{}
	if err := decoder.Decode(options); err != nil {
		return gerrs.Wrap(err, "decode polaris limiter config err")
	}
	log.Infof("plugin %s config:%s", name, convert.ToJSONStr(options))

	if options.Namespace == "" {
		options.Namespace = trpc.GlobalConfig().Global.Namespace
	}
	if options.Service == "" {
		if len(trpc.GlobalConfig().Server.Service) == 0 {
			return errs.New(gerrs.ErrWrongConfig, "empty service")
		}
		options.Service = trpc.GlobalConfig().Server.Service[0].Name
	}
	if options.MaxRetries == 0 {
		options.MaxRetries = p.MaxRetries
	}
	if options.Timeout == 0 {
		options.Timeout = p.Timeout
	}
	return nil
}

// InterceptServer implements the server-side rate limiting interceptor.
func (l *Limiter) InterceptServer(ctx context.Context, req interface{},
	handler filter.ServerHandleFunc) (interface{}, error) {
	hitLimit, err := l.isLimit(ctx)
	if err != nil {
		// Exception, allow the request to proceed
		log.ErrorContextf(ctx, "failed to get quota, err: %s", err)
		return handler(ctx, req)
	}
	if hitLimit {
		return nil, errs.New(ErrLimit, "request rate limit")
	}
	return handler(ctx, req)
}

// Check if rate limit is reached
func (l *Limiter) isLimit(ctx context.Context) (bool, error) {
	defer func() {
		if r := recover(); r != nil {
			log.ErrorContextf(ctx, "get rate limit quota panic:%s", string(debug.Stack()))
		}
	}()
	// Parse plugin configuration
	pluginConfig := gwmsg.GwMessage(ctx).PluginConfig(PluginName)
	if pluginConfig == nil {
		return false, errs.New(gerrs.ErrPluginConfigNotFound, "get no response transformer plugin config")
	}
	options, ok := pluginConfig.(*Options)
	if !ok {
		return false, errs.New(gerrs.ErrPluginConfigNotFound, "invalid response transformer plugin config type")
	}

	quotaReq := api.NewQuotaRequest()
	quotaReq.SetNamespace(options.Namespace)
	// Set the service name, corresponding to the Polaris address, default to the gateway service address
	quotaReq.SetService(options.Service)
	// Set the timeout
	quotaReq.SetTimeout(remainTimeout(ctx, options.Timeout))
	// Set the retry count
	quotaReq.SetRetryCount(options.MaxRetries)
	// Set the request parameters
	quotaReq.SetLabels(getLabels(ctx, options.Labels, options.ParseJSONBody))
	quotaFuture, err := l.API.GetQuota(quotaReq)
	if err != nil {
		return false, gerrs.Wrap(err, "get quota err")
	}
	defer quotaFuture.Release()
	switch qf := quotaFuture.Get(); qf.Code {
	case api.QuotaResultLimited:
		setRspBody(ctx, options.LimitedRspBody)
		return true, nil
	case api.QuotaResultOk:
		fallthrough
	default:
		return false, nil
	}
}

// Set the response body
func setRspBody(ctx context.Context, body string) {
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return
	}
	if len(body) == 0 {
		return
	}
	log.DebugContextf(ctx, "set rate limit body:%s", body)
	fctx.Response.Header.SetContentType("application/json")
	fctx.Response.SetBodyString(body)
}

// Get rate limiting labels
func getLabels(ctx context.Context, labels []string, parseJSONBody bool) map[string]string {
	labelMap := make(map[string]string)
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		// Not an HTTP request
		return labelMap
	}
	// Set the interface name
	// TODO: This needs to be compatible with prefix matching for rate limiting.
	//  The method should be passed from the route item.
	labelMap["method"] = string(fctx.Path())
	for _, label := range labels {
		if val := getParams(fctx, label, parseJSONBody); val != "" {
			labelMap[label] = val
		}
	}
	return labelMap
}

// Get parameters
func getParams(fctx *fasthttp.RequestCtx, label string, parseJSONBody bool) string {
	if val := http.GetString(fctx, label); val != "" {
		return val
	}
	if !parseJSONBody {
		return ""
	}
	return gjson.Get(string(fctx.Request.Body()), label).String()
}

// Calculate remaining timeout
func remainTimeout(ctx context.Context, timeout int) time.Duration {
	timeoutD := time.Millisecond * time.Duration(timeout)
	ctxDeadline, ok := ctx.Deadline()
	if !ok {
		return timeoutD
	}

	if remain := time.Until(ctxDeadline); remain < timeoutD {
		return remain
	}
	return timeoutD
}
