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

// Package routercheck is a plugin for router configuration validation.
package routercheck

import (
	"context"
	"encoding/json"
	"runtime/debug"

	"gopkg.in/yaml.v3"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	cplugin "trpc.group/trpc-go/trpc-gateway/common/plugin"
	"trpc.group/trpc-go/trpc-gateway/core/entity"
	"trpc.group/trpc-go/trpc-gateway/core/router"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
)

const (
	pluginName      = "routercheck"
	checkFailedCode = 1000
)

func init() {
	plugin.Register(pluginName, &Plugin{})
}

// Plugin is the plugin definition.
type Plugin struct {
}

// Type returns the plugin type.
func (p *Plugin) Type() string {
	return cplugin.DefaultType
}

// Setup initializes the plugin.
func (p *Plugin) Setup(string, plugin.Decoder) error {
	// Register the plugin
	filter.Register(pluginName, ServerFilter, nil)
	return nil
}

// Options is the plugin configuration.
type Options struct{}

// CheckConfig validates the plugin configuration and returns the parsed configuration object for use in the
// ServerFilter method.
func (p *Plugin) CheckConfig(_ string, decoder plugin.Decoder) error {
	options := &Options{}
	if err := decoder.Decode(options); err != nil {
		return gerrs.Wrap(err, "decode router check config error")
	}
	return nil
}

// ServerFilter is the server interceptor.
func ServerFilter(ctx context.Context, _ interface{}, _ filter.ServerHandleFunc) (interface{}, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error(ctx, "router check panic: %s, stack: %s", r, debug.Stack())
		}
	}()
	rspBody, err := check(ctx)
	if err != nil {
		return nil, gerrs.Wrap(err, "check router error")
	}
	// Get the request configuration
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return nil, nil
	}
	bodyByte, err := json.Marshal(rspBody)
	if err != nil {
		return nil, gerrs.Wrap(err, "marshal body error")
	}
	fctx.Response.Header.SetContentType("application/json")
	fctx.Response.SetBody(bodyByte)
	return nil, nil
}

// checkConfig validates the configuration.
func check(ctx context.Context) (*rspBody, error) {
	// Get the request configuration
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return nil, nil
	}
	routerConfig := fctx.Request.Body()
	var rf entity.ProxyConfig
	if err := yaml.Unmarshal(routerConfig, &rf); err != nil {
		log.WarnContextf(ctx, "unmarshal router config error: %s", err)
		return &rspBody{
			Code: checkFailedCode,
			Msg:  err.Error(),
		}, nil
	}

	_, err := router.NewFastHTTPRouter().CheckAndInit(ctx, &rf)
	if err != nil {
		log.WarnContextf(ctx, "check and init router config error: %s", err)
		return &rspBody{
			Code: checkFailedCode,
			Msg:  err.Error(),
		}, nil
	}
	return &rspBody{
		Code: 0,
		Msg:  "success",
	}, nil
}

// rspBody is the response structure.
type rspBody struct {
	// Status code, non-zero indicates validation failure
	Code int `json:"code"`
	// Error message, the original error log
	Msg string `json:"msg"`
}
