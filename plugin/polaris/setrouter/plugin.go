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

// Package setrouter sets the router
package setrouter

import (
	"context"
	"runtime/debug"

	"trpc.group/trpc-go/trpc-gateway/common/convert"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	cplugin "trpc.group/trpc-go/trpc-gateway/common/plugin"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
)

const (
	pluginName = "setrouter"
)

func init() {
	plugin.Register(pluginName, &Plugin{})
}

// Plugin defines the plugin
type Plugin struct {
}

// Type gets the plugin type
func (p *Plugin) Type() string {
	return cplugin.DefaultType
}

// Setup initializes the plugin
func (p *Plugin) Setup(_ string, decoder plugin.Decoder) error {
	if err := decoder.Decode(p); err != nil {
		return gerrs.Wrap(err, "decode setrouter global config error")
	}

	setHandler := &SetHandler{}
	filter.Register(pluginName, setHandler.ServerFilter, nil)
	return nil
}

// Options represents the plugin configuration
type Options struct {
	SetName string `yaml:"set_name"` // Set name
}

// CheckConfig validates the plugin configuration and returns the parsed configuration object for use
// in the ServerFilter method
func (p *Plugin) CheckConfig(_ string, decoder plugin.Decoder) error {
	options := &Options{}
	if err := decoder.Decode(options); err != nil {
		return gerrs.Wrap(err, "decode setrouter config error")
	}

	if options.SetName == "" {
		return errs.New(gerrs.ErrInvalidPluginConfig, "empty set name")
	}
	return nil
}

// SetHandler handles the set operation
type SetHandler struct {
}

// ServerFilter is the server interceptor
func (ehl *SetHandler) ServerFilter(ctx context.Context, req interface{},
	handler filter.ServerHandleFunc) (interface{}, error) {
	if err := ehl.setSetName(ctx); err != nil {
		return nil, gerrs.Wrap(err, "set set name error")
	}
	rsp, err := handler(ctx, req)
	return rsp, err
}

// Parse and set the set name
func (ehl *SetHandler) setSetName(ctx context.Context) error {
	defer func() {
		if r := recover(); r != nil {
			log.Error(ctx, "set router panic: %s, stack: %s", r, debug.Stack())
		}
	}()

	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return nil
	}
	// Parse plugin configuration
	pluginConfig := gwmsg.GwMessage(ctx).PluginConfig(pluginName)
	if pluginConfig == nil {
		return errs.New(gerrs.ErrWrongConfig, "no set router config found")
	}
	options, ok := pluginConfig.(*Options)
	if !ok {
		return errs.New(gerrs.ErrWrongConfig, "invalid set router config")
	}

	log.DebugContextf(ctx, "setrouter_config: %s", convert.ToJSONStr(options))

	gwmsg.GwMessage(ctx).WithTRPCClientOpts([]client.Option{
		client.WithCalleeSetName(options.SetName),
		client.WithTarget(""),
	})
	return nil
}
