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

// Package mocking provides mock responses
package mocking

import (
	"context"
	"math/rand"
	"runtime/debug"
	"time"

	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/convert"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	cplugin "trpc.group/trpc-go/trpc-gateway/common/plugin"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
)

const (
	pluginName = "mocking"
	// Mock response body error
	errMocking  = 10006
	clientIPKey = "client_ip"
)

func init() {
	plugin.Register(pluginName, &Plugin{})
}

// Plugin defines the plugin
type Plugin struct {
}

// Type returns the plugin type
func (p *Plugin) Type() string {
	return cplugin.DefaultType
}

// Setup initializes the plugin
func (p *Plugin) Setup(string, plugin.Decoder) error {
	// Register the plugin
	filter.Register(pluginName, ServerFilter, nil)
	if err := gerrs.Register(errMocking, fasthttp.StatusServiceUnavailable); err != nil {
		return gerrs.Wrap(err, "register mocking plugin code err")
	}
	return nil
}

// Options represents the plugin configuration
type Options struct {
	// Delay in milliseconds before returning the response, default is 0
	Delay int `yaml:"delay"`
	// HTTP status code of the response, default is 200
	ResponseStatus int `yaml:"response_status"`
	// Content-Type header of the response, default is "application/json"
	ContentType string `yaml:"content_type"`
	// Body of the response
	ResponseExample string `yaml:"response_example"`
	// When set to true, adds the response header "x-mock-by: tRPC-Gateway". When set to false, the header is not added.
	WithMockHeader bool `yaml:"with_mock_header"`
	// Mock traffic ratio, in percentage. For example, if it is one in ten thousand, fill in: 0.01. The default is
	// full mock.
	Scale float64 `yaml:"scale" json:"scale"`
	// Hash key for mock traffic, providing the ability to perform grayscale based on request parameters.
	HashKey string `yaml:"hash_key" json:"hash_key"`
}

// CheckConfig validates the plugin configuration and returns the parsed configuration object. Used in the ServerFilter
// method for parsing.
func (p *Plugin) CheckConfig(_ string, decoder plugin.Decoder) error {
	options := &Options{}
	if err := decoder.Decode(options); err != nil {
		return gerrs.Wrap(err, "decode dev env config err")
	}
	if options.ResponseExample == "" {
		return errs.New(gerrs.ErrInvalidPluginConfig, "response example empty")
	}
	if options.ResponseStatus == 0 {
		options.ResponseStatus = fasthttp.StatusOK
	}
	if options.ContentType == "" {
		options.ContentType = "application/json"
	}
	if options.Scale <= 0 {
		options.Scale = 100
	}
	return nil
}

// ServerFilter is the server interceptor
func ServerFilter(ctx context.Context, req interface{}, handler filter.ServerHandleFunc) (interface{}, error) {
	mock, err := mockRsp(ctx)
	if err != nil {
		return nil, errs.New(errMocking, "mocking rsp err")
	}
	if mock {
		return nil, nil
	}
	return handler(ctx, req)
}

// Override the forwarding environment
func mockRsp(ctx context.Context) (bool, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error(ctx, "mocking panic:%s,stack:%s", r, debug.Stack())
		}
	}()

	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return false, nil
	}
	// Parse the plugin configuration
	pluginConfig := gwmsg.GwMessage(ctx).PluginConfig(pluginName)
	if pluginConfig == nil {
		return false, errs.New(gerrs.ErrWrongConfig, "get no mocking config")
	}
	options, ok := pluginConfig.(*Options)
	if !ok {
		log.ErrorContextf(ctx, "invalid mocking config")
		return false, errs.New(gerrs.ErrWrongConfig, "invalid mocking config")
	}
	// Check if grayscale is hit
	if !isHitGrey(fctx, options) {
		return false, nil
	}

	// Add delay if specified
	if options.Delay != 0 {
		time.Sleep(time.Millisecond * time.Duration(options.Delay))
	}

	// Set response headers and body based on options
	fctx.Response.Header.SetContentType(options.ContentType)
	fctx.Response.SetBodyString(options.ResponseExample)
	fctx.Response.SetStatusCode(options.ResponseStatus)

	// Add mock header if specified
	if options.WithMockHeader {
		fctx.Response.Header.Set("X-Mock-By", http.GatewayName)
	}
	return true, nil
}

// Check if grayscale is hit
func isHitGrey(fctx *fasthttp.RequestCtx, options *Options) bool {
	if options.Scale == 100 || options.Scale == 0 {
		return true
	}
	if options.Scale > getRandNum(fctx, options.HashKey) {
		return true
	}
	return false
}

// Get a random number
func getRandNum(fctx *fasthttp.RequestCtx, hashKey string) float64 {
	if hashKey == "" {
		// Random number seed has been set during gateway initialization
		return rand.Float64() * 100
	}
	val := getParam(fctx, hashKey)
	if val != "" {
		return float64(convert.Fnv32(val) % uint32(100))
	}
	// Definitely not hit
	return 100
}

// Get parameter
func getParam(fctx *fasthttp.RequestCtx, reqKey string) string {
	if reqKey == clientIPKey {
		return http.GetClientIP(fctx)
	}
	return http.GetString(fctx, reqKey)
}
