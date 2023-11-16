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

// Package devenv provides the ability to configure the development environment
package devenv

import (
	"context"
	"fmt"
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
)

const (
	pluginName = "devenv"
	// Development environment forwarding error
	errDevEnv = 10007
	// Parameter for specifying the environment
	requestDomainKey = "request-domain"
)

func init() {
	plugin.Register(pluginName, &Plugin{})
}

// Plugin is plugin definition
type Plugin struct {
	// Parameter name that identifies the environment
	EnvKey string `yaml:"env_key"`
}

// Type return plugin type
func (p *Plugin) Type() string {
	return cplugin.DefaultType
}

// Setup Plugin initialization plugin
func (p *Plugin) Setup(_ string, decoder plugin.Decoder) error {
	if err := decoder.Decode(p); err != nil {
		return gerrs.Wrap(err, "decode devenv global config err")
	}
	if p.EnvKey == "" {
		p.EnvKey = requestDomainKey
	}

	// Register the plugin
	envHandler := &EnvHandler{EnvKey: p.EnvKey}
	filter.Register(pluginName, envHandler.ServerFilter, nil)
	if err := gerrs.Register(errDevEnv, fasthttp.StatusServiceUnavailable); err != nil {
		return gerrs.Wrap(err, "register dev env code err")
	}
	return nil
}

// Options is plugin configuration
type Options struct {
	EnvList []*EnvConfig `yaml:"env_list"`
}

// EnvConfig is environment configuration
type EnvConfig struct {
	// Personal environment name, usually the English name of the enterprise micro
	RequestDomain string `yaml:"request_domain"`
	// Whether to enable
	Disable              bool `yaml:"disable"`
	client.BackendConfig `yaml:",inline"`
}

// CheckConfig validate the plugin configuration and return the parsed configuration object.
// Used in the ServerFilter method for parsing.
func (p *Plugin) CheckConfig(_ string, decoder plugin.Decoder) error {
	options := &Options{}
	if err := decoder.Decode(options); err != nil {
		return gerrs.Wrap(err, "decode dev env config err")
	}

	for _, config := range options.EnvList {
		if config.RequestDomain == "" {
			return errs.Newf(gerrs.ErrWrongConfig, "empty dev env name:%s", convert.ToJSONStr(config))
		}
	}
	return nil
}

// EnvHandler is environment handler object
type EnvHandler struct {
	EnvKey string
}

// ServerFilter executes plugin logic
func (ehl *EnvHandler) ServerFilter(ctx context.Context, req interface{},
	handler filter.ServerHandleFunc) (interface{}, error) {
	if err := ehl.overWriteEnv(ctx); err != nil {
		return nil, errs.New(errDevEnv, "invalid dev env name")
	}
	rsp, err := handler(ctx, req)
	setEnvHeader(ctx)
	if err != nil {
		return nil, err
	}
	return rsp, nil
}

// overWriteEnv overwrite the forwarded environment
func (ehl *EnvHandler) overWriteEnv(ctx context.Context) error {
	defer func() {
		if r := recover(); r != nil {
			log.Error(ctx, "replay handle panic:%s,stack:%s", r, debug.Stack())
		}
	}()

	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return nil
	}
	// Parse the plugin configuration
	pluginConfig := gwmsg.GwMessage(ctx).PluginConfig(pluginName)
	if pluginConfig == nil {
		return errs.New(gerrs.ErrWrongConfig, "get no devenv config")
	}
	options, ok := pluginConfig.(*Options)
	if !ok {
		log.ErrorContextf(ctx, "invalid devenv config")
		return errs.New(gerrs.ErrWrongConfig, "invalid devenv config")
	}

	log.InfoContextf(ctx, "dev_env_config:%s", convert.ToJSONStr(options))
	// Get the environment identifier request header sent by the client
	devEnvName := http.GetString(fctx, ehl.EnvKey)
	if len(devEnvName) == 0 {
		return nil
	}

	for _, config := range options.EnvList {
		if config.Disable {
			continue
		}
		log.InfoContextf(ctx, "dev_env_name:%s,config:%s", devEnvName, config.RequestDomain)
		if devEnvName != config.RequestDomain {
			log.InfoContextf(ctx, "dev_env_name ok:%s,config:%s", devEnvName, config.RequestDomain)
			continue
		}
		// Overwrite the target configuration
		cliConf := gwmsg.GwMessage(ctx).TargetService()
		if cliConf == nil {
			return errs.New(gerrs.ErrContextNoServiceVal, "get no target service")
		}
		// Make a copy
		newClient := &client.BackendConfig{}
		copyCommonClient(cliConf, newClient)

		if config.ServiceName != "" {
			newClient.ServiceName = config.ServiceName
		}
		if config.Target != "" {
			newClient.Target = config.Target
		}

		if config.Namespace != "" {
			newClient.Namespace = config.Namespace
		}

		if config.EnvName != "" {
			newClient.EnvName = config.EnvName
		}
		if config.SetName != "" {
			log.InfoContextf(ctx, "set_name ok:%s", config.SetName)
			newClient.SetName = config.SetName
			// Use set addressing, cannot set the target field, use service name to specify the North Star address
			newClient.Target = ""
		}

		newClient.DisableServiceRouter = config.DisableServiceRouter
		gwmsg.GwMessage(ctx).WithTargetService(newClient)
		return nil
	}
	return nil
}

// Set the environment response header
func setEnvHeader(ctx context.Context) {
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return
	}
	target := gwmsg.GwMessage(ctx).TargetService()
	fctx.Response.Header.Set("x-proxy-env",
		fmt.Sprintf("target=%s&namespace=%s&env_name=%s&set_name=%s",
			target.Target, target.Namespace, target.EnvName, target.SetName))
	fctx.Response.Header.Set("x-upstream-ip", gwmsg.GwMessage(ctx).UpstreamAddr())
}

// copyCommonClient copy client info
func copyCommonClient(client, newClient *client.BackendConfig) {
	newClient.ServiceName = client.ServiceName
	newClient.Namespace = client.Namespace
	newClient.EnvName = client.EnvName
	newClient.Network = client.Network
	newClient.Protocol = client.Protocol
	newClient.Target = client.Target
	newClient.Timeout = client.Timeout
	newClient.SetName = client.SetName
	newClient.CalleeMetadata = client.CalleeMetadata
	newClient.DisableServiceRouter = client.DisableServiceRouter
}
