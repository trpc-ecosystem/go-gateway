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

// Package metarouter Polaris Metadata Router Plugin
package metarouter

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
	pluginName = "metarouter"
)

func init() {
	plugin.Register(pluginName, &Plugin{})
}

// Plugin Plugin definition
type Plugin struct {
	// Parameter name that identifies the environment
	EnvKey string `yaml:"env_key"`
}

// Type Get plugin type
func (p *Plugin) Type() string {
	return cplugin.DefaultType
}

// Setup Plugin initialization
func (p *Plugin) Setup(_ string, decoder plugin.Decoder) error {
	if err := decoder.Decode(p); err != nil {
		return gerrs.Wrap(err, "decode metarouter global config err")
	}

	// Register plugin
	envHandler := &EnvHandler{}
	filter.Register(pluginName, envHandler.ServerFilter, nil)
	return nil
}

// Options Plugin configuration
type Options struct {
	MetaKeys []string `yaml:"meta_key_list"`
}

// CheckConfig Validate plugin configuration and return the parsed configuration object. Used in ServerFilter
// method for parsing.
func (p *Plugin) CheckConfig(_ string, decoder plugin.Decoder) error {
	options := &Options{}
	if err := decoder.Decode(options); err != nil {
		return gerrs.Wrap(err, "decode dev env config err")
	}

	if len(options.MetaKeys) == 0 {
		return errs.New(gerrs.ErrInvalidPluginConfig, "empty meta keys")
	}
	for _, key := range options.MetaKeys {
		if key == "" {
			return errs.New(gerrs.ErrInvalidPluginConfig, "meta key can not be empty string")
		}
	}
	return nil
}

// EnvHandler Environment handler object
type EnvHandler struct {
}

// ServerFilter Server interceptor
func (ehl *EnvHandler) ServerFilter(ctx context.Context, req interface{},
	handler filter.ServerHandleFunc) (interface{}, error) {
	if err := ehl.parseAndSetMetadata(ctx); err != nil {
		log.ErrorContextf(ctx, "set polaris metadata err")
	}
	rsp, err := handler(ctx, req)
	return rsp, err
}

// Parse and set metadata
func (ehl *EnvHandler) parseAndSetMetadata(ctx context.Context) error {
	defer func() {
		if r := recover(); r != nil {
			log.Error(ctx, "meta router panic:%s,stack:%s", r, debug.Stack())
		}
	}()

	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return nil
	}
	// Parse plugin configuration
	pluginConfig := gwmsg.GwMessage(ctx).PluginConfig(pluginName)
	if pluginConfig == nil {
		return errs.New(gerrs.ErrWrongConfig, "get no metarouter config")
	}
	options, ok := pluginConfig.(*Options)
	if !ok {
		log.ErrorContextf(ctx, "invalid metarouter config")
		return errs.New(gerrs.ErrWrongConfig, "invalid metarouter config")
	}
	// Check if service routing is disabled
	if !gwmsg.GwMessage(ctx).TargetService().DisableServiceRouter {
		return errs.New(gerrs.ErrWrongConfig, "polaris meta router need disable service router")
	}
	log.DebugContextf(ctx, "metarouter_config:%s", convert.ToJSONStr(options))
	var opts []client.Option
	for _, key := range options.MetaKeys {
		val := http.GetString(fctx, key)
		if val == "" {
			log.DebugContextf(ctx, "get no value,key:%s", key)
			continue
		}
		opts = append(opts, client.WithCalleeMetadata(key, val))
	}
	gwmsg.GwMessage(ctx).WithTRPCClientOpts(opts)
	return nil
}
