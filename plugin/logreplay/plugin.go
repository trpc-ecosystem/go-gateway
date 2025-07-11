//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

// Package logreplay Log Replay Plugin
package logreplay

import (
	"context"
	"fmt"
	"math/rand"
	"mime"
	"net/url"
	"runtime/debug"
	"time"

	"github.com/tidwall/sjson"
	"github.com/valyala/fasthttp"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	cplugin "trpc.group/trpc-go/trpc-gateway/common/plugin"
	"trpc.group/trpc-go/trpc-gateway/core/service/fhttp"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
)

const (
	pluginName    = "logreplay"
	originRspBody = "origin_rsp_body"
	// Default timeout in milliseconds
	defaultTimeout = 500
)

// DefaultConnPoolSize default connection pool size
var DefaultConnPoolSize = 1000

func init() {
	plugin.Register(pluginName, &Plugin{})
}

// Plugin plugin definition
type Plugin struct{}

// Type get plugin type
func (p *Plugin) Type() string {
	return cplugin.DefaultType
}

// Setup plugin initialization
func (p *Plugin) Setup(string, plugin.Decoder) error {
	lr, err := New()
	if err != nil {
		return gerrs.Wrap(err, "new log replay err")
	}
	// Register the plugin
	filter.Register(pluginName, lr.ServerFilter, nil)
	return nil
}

// LogReplay traffic replay
type LogReplay struct {
	ConnPool fhttp.Pool
}

// New creates a new log replay
var New = func() (*LogReplay, error) {
	l := &LogReplay{
		ConnPool: fhttp.NewConnPool(DefaultConnPoolSize),
	}
	return l, nil
}

// Options plugin configuration
type Options struct {
	// Traffic replay ratio, in percentage. For example, 0.01 represents 1 in 10,000.
	Scale float64 `yaml:"scale" json:"scale"`
	// Whether to pass through the original response body. If true, the original response body will be passed through.
	// For POST requests:
	// Content-Type: application/json, the original response body will be added as a string field "origin_rsp_body"
	// in the original JSON request body.
	// Content-Type: application/x-www-form-urlencoded, the original response body will be added as a URL-encoded
	// field "origin_rsp_body" in the original query request body.
	// For other requests (GET, etc.), the original request will be appended to the query as a URL-encoded
	// field "origin_rsp_body".
	PassThroughResponse bool `yaml:"pass_through_response"`
	// Timeout in milliseconds; default is 500 milliseconds.
	Timeout int `yaml:"timeout"`
}

// CheckConfig validates the plugin configuration and returns a parsed configuration object with the correct type.
// Used in the ServerFilter method for parsing.
func (p *Plugin) CheckConfig(_ string, decoder plugin.Decoder) error {
	options := &Options{}
	if err := decoder.Decode(options); err != nil {
		return gerrs.Wrap(err, "decode polaris limiter config err")
	}
	if options.Scale < 0 || options.Scale > 100 {
		return errs.Newf(gerrs.ErrWrongConfig, "invalid log replay scale:%v", options.Scale)
	}
	if options.Timeout == 0 {
		options.Timeout = defaultTimeout
	}
	return nil
}

// ServerFilter server interceptor
func (lr *LogReplay) ServerFilter(ctx context.Context, req interface{},
	handler filter.ServerHandleFunc) (interface{}, error) {
	rsp, err := handler(ctx, req)
	if rerr := lr.handle(ctx); rerr != nil {
		log.ErrorContextf(ctx, "log replay err:%s", rerr)
	}
	return rsp, err
}

func (lr *LogReplay) handle(ctx context.Context) error {
	defer func() {
		if r := recover(); r != nil {
			log.Error(ctx, "replay handle panic:%s,stack:%s", r, debug.Stack())
		}
	}()
	// Parse plugin configuration
	pluginConfig := gwmsg.GwMessage(ctx).PluginConfig(pluginName)
	if pluginConfig == nil {
		return errs.New(gerrs.ErrPluginConfigNotFound, "get no log replay config")
	}
	options, ok := pluginConfig.(*Options)
	if !ok {
		return errs.New(gerrs.ErrPluginConfigNotFound, "invalid log replay config")
	}
	// Random number seed has been set during gateway initialization
	if options.Scale == 0 || options.Scale < rand.Float64()*100 {
		return nil
	}

	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return nil
	}
	// Copy the request
	newReq, err := lr.CopyRequest(ctx, fctx, options)
	if err != nil {
		return gerrs.Wrap(err, "copy request err")
	}
	newCtx := trpc.CloneContext(ctx)
	go func() {
		if err := lr.Replay(newCtx, newReq, options); err != nil {
			log.ErrorContextf(newCtx, "log replay err:%s,path:%s", err, newReq.URI().Path())
		}
	}()
	return nil
}

// Replay replays the request
func (lr *LogReplay) Replay(ctx context.Context, newReq *fasthttp.Request, opts *Options) error {
	defer func() {
		if r := recover(); r != nil {
			log.Error(ctx, "log replay panic:%s,stack:%s", r, debug.Stack())
		}
		fasthttp.ReleaseRequest(newReq)
	}()

	// Get local IP
	if len(trpc.GlobalConfig().Server.Service) == 0 {
		return errs.New(gerrs.ErrWrongConfig, "get no service config")
	}
	host := fmt.Sprintf("%s:%v", trpc.GlobalConfig().Server.Service[0].IP,
		trpc.GlobalConfig().Server.Service[0].Port)
	// If host is empty, set it to the IP:port of the request, otherwise it will throw an error of empty host
	if len(newReq.URI().Host()) == 0 {
		newReq.URI().SetHost(host)
	}
	proxyClient, err := lr.ConnPool.Get(host)
	if err != nil {
		return gerrs.Wrap(err, "get replay conn err")
	}
	proxyClient.Addr = host
	defer func() {
		_ = lr.ConnPool.Put(proxyClient)
	}()

	// Get timeout
	timeout := time.Duration(opts.Timeout) * time.Millisecond
	proxyClient.MaxIdleConnDuration = timeout
	proxyClient.ReadTimeout = timeout
	proxyClient.WriteTimeout = timeout
	otherResp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(otherResp)
	err = proxyClient.Do(newReq, otherResp)
	// Solve the issue of closing the connection when reusing the connection
	if err == fasthttp.ErrConnectionClosed {
		log.ErrorContextf(ctx, "replay conn closed err:%s", err)
		err = proxyClient.Do(newReq, otherResp)
	}
	if err != nil {
		return gerrs.Wrap(err, "do replay request err")
	}
	log.DebugContextf(ctx, "resp stats:%v, req:%s,body:%s",
		otherResp.StatusCode(), string(newReq.Body()), string(otherResp.Body()))
	return nil
}

// CopyRequest copies the request
func (lr *LogReplay) CopyRequest(ctx context.Context, fctx *fasthttp.RequestCtx,
	opts *Options) (*fasthttp.Request, error) {
	newReq := fasthttp.AcquireRequest()
	fctx.Request.CopyTo(newReq)
	newReq.URI().SetPath(fmt.Sprintf("%s_replay", fctx.Path()))
	// Append the original response body to the copied request parameters for diffing in the new interface
	if opts.PassThroughResponse {
		if err := setOriginRspBody(ctx, fctx, newReq); err != nil {
			fasthttp.ReleaseRequest(newReq)
			return nil, gerrs.Wrap(err, "set origin rsp err")
		}
	}
	return newReq, nil
}

// Set the original response body to the copied request
func setOriginRspBody(_ context.Context, fctx *fasthttp.RequestCtx, newReq *fasthttp.Request) error {
	if string(fctx.Request.Header.Method()) != fasthttp.MethodPost {
		newReq.URI().QueryArgs().Set(originRspBody, url.QueryEscape(string(fctx.Response.Body())))
		return nil
	}
	bT, _, err := mime.ParseMediaType(string(fctx.Request.Header.ContentType()))
	if err != nil {
		return gerrs.Wrap(err, "parse content type err")
	}
	if bT == "application/json" {
		newReqBody, err := sjson.SetBytes(newReq.Body(), originRspBody, string(fctx.Response.Body()))
		if err != nil {
			return gerrs.Wrap(err, "set origin response body err")
		}
		newReq.SetBody(newReqBody)
		return nil
	}
	if bT == "application/x-www-form-urlencoded" {
		newReq.PostArgs().Set(originRspBody, url.QueryEscape(string(fctx.Response.Body())))
		// Note: It is necessary to reset the body here, otherwise if the body has a value, it will cause the parameter
		// setting to fail
		// Related issue: https://github.com/erikdubbelboer/fasthttp/issues/17
		newReq.SetBody(newReq.PostArgs().QueryString())
		return nil
	}
	newReq.URI().QueryArgs().Set(originRspBody, url.QueryEscape(string(fctx.Response.Body())))
	return nil
}
