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

// Package canaryrouter Canary Router
package canaryrouter

import (
	"context"
	"math/rand"
	"runtime/debug"

	"github.com/valyala/fasthttp"
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
	"trpc.group/trpc-go/trpc-naming-polarismesh/servicerouter"
)

const (
	pluginName  = "canaryrouter"
	clientIPKey = "client_ip"
)

func init() {
	plugin.Register(pluginName, &Plugin{})
}

// Plugin Plugin definition
type Plugin struct {
}

// Type Get plugin type
func (p *Plugin) Type() string {
	return cplugin.DefaultType
}

// Setup Plugin initialization
func (p *Plugin) Setup(_ string, decoder plugin.Decoder) error {
	if err := decoder.Decode(p); err != nil {
		return gerrs.Wrap(err, "decode canaryrouter global config err")
	}

	// Register plugin
	envHandler := &CanaryHandler{}
	filter.Register(pluginName, envHandler.ServerFilter, nil)
	return nil
}

// Options Plugin configuration
type Options struct {
	// ReqKey Canary key
	ReqKey string `yaml:"request_key"`
	// Values Canary values
	Values []string `yaml:"values"`
	// valueMap Parsed values
	valueMap map[string]bool `yaml:"-"`
	// Canary traffic ratio, in percentage, e.g., 0.01 for one in ten thousand
	Scale float64 `yaml:"scale" json:"scale"`
	// Canary traffic hash key
	HashKey string `yaml:"hash_key" json:"hash_key"`
	// CanaryTagVal PolarisMesh Canary tag value, used with 123 to customize Canary tags, default is 1
	CanaryTagVal string `yaml:"canary_tag_val"`
}

// CheckConfig Validate plugin configuration and return the parsed configuration object. Used in ServerFilter
// method for parsing.
func (p *Plugin) CheckConfig(_ string, decoder plugin.Decoder) error {
	options := &Options{}
	if err := decoder.Decode(options); err != nil {
		return gerrs.Wrap(err, "decode canary router config err")
	}
	if options.Scale < 0 || options.Scale > 100 {
		return errs.Newf(gerrs.ErrWrongConfig, "invalid canary scale:%v", options.Scale)
	}
	options.valueMap = make(map[string]bool)
	for _, key := range options.Values {
		options.valueMap[key] = true
	}
	if options.CanaryTagVal == "" {
		options.CanaryTagVal = "1"
	}
	return nil
}

// CanaryHandler Canary handler
type CanaryHandler struct {
}

// ServerFilter Server interceptor
func (ehl *CanaryHandler) ServerFilter(ctx context.Context, req interface{},
	handler filter.ServerHandleFunc) (interface{}, error) {
	if err := ehl.setCanary(ctx); err != nil {
		log.ErrorContextf(ctx, "set canary err:%s", err)
	}
	rsp, err := handler(ctx, req)
	return rsp, err
}

// Parse and set Canary flag
func (ehl *CanaryHandler) setCanary(ctx context.Context) error {
	defer func() {
		if r := recover(); r != nil {
			log.Error(ctx, "canary router panic:%s,stack:%s", r, debug.Stack())
		}
	}()

	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return nil
	}
	// Parse plugin configuration
	pluginConfig := gwmsg.GwMessage(ctx).PluginConfig(pluginName)
	if pluginConfig == nil {
		return errs.New(gerrs.ErrWrongConfig, "get no canary router config")
	}
	options, ok := pluginConfig.(*Options)
	if !ok {
		return errs.New(gerrs.ErrWrongConfig, "invalid canary router config")
	}

	log.DebugContextf(ctx, "canary_router_config:%s", convert.ToJSONStr(options))
	hit, err := ehl.isHitCanary(ctx, fctx, options)
	if err != nil {
		return gerrs.Wrap(err, "check canary hit status err")
	}
	if hit {
		log.DebugContext(ctx, "set canary")
		gwmsg.GwMessage(ctx).WithTRPCClientOpts([]client.Option{
			servicerouter.WithCanary(options.CanaryTagVal),
		})
		return nil
	}
	return nil
}

// Check if Canary is hit
func (ehl *CanaryHandler) isHitCanary(ctx context.Context, fctx *fasthttp.RequestCtx, options *Options) (bool, error) {
	// Full traffic
	if options.Scale == 100 {
		return true, nil
	}

	// Hit whitelist
	if val := getParam(fctx, options.ReqKey); val != "" {
		if exist := options.valueMap[val]; exist {
			log.DebugContextf(ctx, "set canary value:%s", val)
			return true, nil
		}
	}

	// Hit gray traffic
	if options.Scale != 0 && options.Scale > ehl.getRandNum(fctx, options.HashKey) {
		return true, nil
	}
	return false, nil
}

// Get random number
func (ehl *CanaryHandler) getRandNum(fctx *fasthttp.RequestCtx, hashKey string) float64 {
	if hashKey == "" {
		return rand.Float64() * 100
	}
	val := getParam(fctx, hashKey)
	if val != "" {
		return float64(convert.Fnv32(val) % uint32(100))
	}
	// Definitely not hit
	return 100
}

func getParam(fctx *fasthttp.RequestCtx, reqKey string) string {
	if reqKey == "" {
		return ""
	}
	if reqKey == clientIPKey {
		return http.GetClientIP(fctx)
	}
	return http.GetString(fctx, reqKey)
}
