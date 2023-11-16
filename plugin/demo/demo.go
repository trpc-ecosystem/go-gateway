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

// Package demo is an example of developing a gateway plugin. You can refer to this example to develop your own plugin.
package demo

import (
	"context"
	"errors"
	"runtime/debug"

	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/convert"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	cplugin "trpc.group/trpc-go/trpc-gateway/common/plugin"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"
)

// demoFailedCode Custom error code for the plugin
const demoFailedCode = 40001

// Plugin Signature verification plugin
type Plugin struct{}

// Options Configuration for the authentication plugin. Here, a separate configuration object needs to be defined.
type Options struct {
	// Parameter name for receiving suid in the proxy interface
	SUIDName string `yaml:"suid_name"`
}

const (
	// pluginName Plugin name
	pluginName = "demo"
)

func init() {
	plugin.Register(pluginName, &Plugin{})
}

// Type Get the plugin type
func (p *Plugin) Type() string {
	return cplugin.DefaultType
}

// DependsOn Dependent plugins, such as: config-etcd
func (p *Plugin) DependsOn() []string {
	return []string{}
}

// Setup Plugin initialization
func (p *Plugin) Setup(string, plugin.Decoder) error {
	filter.Register(pluginName, ServerFilter, nil)
	// Register the mapping of plugin's custom error code to http status code
	if err := gerrs.Register(demoFailedCode, fasthttp.StatusUnauthorized); err != nil {
		return errors.New("register demo failed code err")
	}
	return nil
}

// CheckConfig verifies the plugin configuration and returns the parsed configuration object. Used in the ServerFilter
// method for parsing.
func (p *Plugin) CheckConfig(_ string, decoder plugin.Decoder) error {
	demoConfig := &Options{}
	if err := decoder.Decode(demoConfig); err != nil {
		return gerrs.Wrap(err, "decode demo config err")
	}

	// Perform validation and initialization of plugin parameters;
	// Note that if it is an append operation for an array, make sure to deduplicate, otherwise it will cause duplicate
	// parameters during global execution
	if demoConfig.SUIDName == "" {
		demoConfig.SUIDName = "suid"
	}
	return nil
}

// ServerFilter Server-side interceptor
func ServerFilter(ctx context.Context, req interface{}, handler filter.ServerHandleFunc) (interface{}, error) {
	// Perform operations on the request body
	if err := preFunc(ctx); err != nil {
		return nil, gerrs.Wrap(err, "pre func err")
	}
	// Execute forwarding
	rsp, err := handler(ctx, req)
	if err != nil {
		return nil, err
	}
	// Perform operations on the response body and response headers
	if err = postFunc(ctx); err != nil {
		return nil, gerrs.Wrap(err, "post func err")
	}
	return rsp, nil
}

func preFunc(ctx context.Context) error {
	defer func() {
		// Add panic recovery to prevent interface exceptions caused by plugin panics
		if r := recover(); r != nil {
			log.ErrorContextf(ctx, "demo handle panic:%s", string(debug.Stack()))
		}
	}()
	// Get the plugin configuration for the route item
	pluginConfig := gwmsg.GwMessage(ctx).PluginConfig(pluginName)
	if pluginConfig == nil {
		return errs.New(gerrs.ErrPluginConfigNotFound, "get no demo plugin config")
	}
	demoConfig, ok := pluginConfig.(*Options)
	if !ok {
		return errs.New(gerrs.ErrPluginConfigNotFound, "invalid demo plugin config type")
	}

	// Modify the request content, such as extracting the field "business_key" from the JSON request body and putting
	// it in the request header
	//fctx := http.RequestContext(ctx)
	//if fctx == nil {
	//	// Not an HTTP request
	//	return nil
	//}
	//// Not a JSON request
	//if !strings.Contains(string(fctx.Request.Header.ContentType()), "json") {
	//	return nil
	//}
	//// Use github.com/tidwall/gjson to manipulate JSON
	//businessVal := gjson.GetBytes(fctx.Request.Body(), "business_key")
	//fctx.Request.Header.Set("business_key", businessVal.String())

	log.InfoContextf(ctx, "demo plugin config:%s", convert.ToJSONStr(demoConfig))
	// Perform some intervention on the request content
	return nil
}

func postFunc(ctx context.Context) error {
	defer func() {
		// Add panic recovery to prevent interface exceptions caused by plugin panics
		if r := recover(); r != nil {
			log.ErrorContextf(ctx, "demo handle panic:%s", string(debug.Stack()))
		}
	}()
	// Get the plugin configuration for the route item
	pluginConfig := gwmsg.GwMessage(ctx).PluginConfig(pluginName)
	if pluginConfig == nil {
		return errs.New(gerrs.ErrPluginConfigNotFound, "get no demo plugin config")
	}
	demoConfig, ok := pluginConfig.(*Options)
	if !ok {
		return errs.New(gerrs.ErrPluginConfigNotFound, "invalid demo plugin config type")
	}
	// Get the original response content from the upstream
	gwMsg := gwmsg.GwMessage(ctx)
	targetService := gwMsg.TargetService()
	if targetService.Protocol == trpc.ProtocolName {
		if gwMsg.UpstreamRspHead() != nil && gwMsg.UpstreamRspHead().(*trpcpb.ResponseProtocol) != nil {
			md := gwMsg.UpstreamRspHead().(*trpcpb.ResponseProtocol).TransInfo
			log.DebugContextf(ctx, "server metadata:%s\n", convert.ToJSONStr(md))
		}
	}

	log.InfoContextf(ctx, "demo plugin config:%s", convert.ToJSONStr(demoConfig))
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		// Not an HTTP request
		return nil
	}
	log.InfoContextf(ctx, "demo plugin config:%s", convert.ToJSONStr(fctx.Response.Header.String()))
	if !fctx.Response.IsBodyStream() {
		log.InfoContextf(ctx, "demo plugin config:%s", string(fctx.Response.Body()))
	}
	// Perform some intervention on the response content
	// Note: Do not reference the fctx object in a goroutine because it will be reused by fasthttp after the request
	// returns.
	// You can use the fasthttp.Request.CopyTo method to make a copy.
	// Reference: https://github.com/valyala/fasthttp/issues/146
	return nil
}
