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

package devenv_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/plugin"
	"trpc.group/trpc-go/trpc-gateway/plugin/devenv"
	"trpc.group/trpc-go/trpc-go/client"
	tplugin "trpc.group/trpc-go/trpc-go/plugin"
)

func TestPlugin_CheckConfig(t *testing.T) {
	p := &devenv.Plugin{}
	tDecoder := &tplugin.YamlNodeDecoder{
		Node: &yaml.Node{},
	}
	err := p.Setup("", tDecoder)
	assert.Nil(t, err)
	opts := &devenv.Options{
		EnvList: []*devenv.EnvConfig{
			{
				RequestDomain: "",
				BackendConfig: client.BackendConfig{
					Namespace:   "Development",
					ServiceName: "yyy",
					SetName:     "set.sz.1",
					EnvName:     "pre",
					Target:      "polaris://yyy",
				},
			},
			{
				RequestDomain: "test",
				Disable:       true,
				BackendConfig: client.BackendConfig{
					Namespace:   "Development",
					ServiceName: "yyy",
					SetName:     "set.sz.1",
					EnvName:     "pre",
					Target:      "polaris://yyy",
				},
			},
		},
	}

	// Configuration validation failed
	decoder := &plugin.PropsDecoder{Props: opts}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)

	// Configuration validation succeeded
	opts.EnvList[0].RequestDomain = "pre"
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)

	// Plugin execution succeeded
	envHandler := devenv.EnvHandler{EnvKey: "request-domain"}
	_, err = envHandler.ServerFilter(context.Background(), nil, func(ctx context.Context,
		req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Forwarding to baseline environment
	fctx := &fasthttp.RequestCtx{}
	ctx := http.WithRequestContext(context.Background(), fctx)
	ctx, msg := gwmsg.WithNewGWMessage(ctx)
	msg.WithTargetService(&client.BackendConfig{
		ServiceName: "xxxx",
		Namespace:   "Production",
		EnvName:     "formal",
		SetName:     "set.tj.0",
		Target:      "polaris://xxx",
	})
	msg.WithPluginConfig("devenv", opts)
	_, err = envHandler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// request-domain did not match
	fctx.Request.Header.Set("request-domain", "test")
	msg.WithTargetService(&client.BackendConfig{
		ServiceName: "xxxx",
		Namespace:   "Production",
		EnvName:     "formal",
		SetName:     "set.tj.0",
		Target:      "polaris://xxx",
	})
	msg.WithPluginConfig("devenv", opts)
	_, err = envHandler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Forwarding to preview environment
	fctx.Request.Header.Set("request-domain", "pre")
	_, err = envHandler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	targetService := msg.TargetService()
	assert.Equal(t, "yyy", targetService.ServiceName)
	assert.Equal(t, "Development", targetService.Namespace)
	assert.Equal(t, "pre", targetService.EnvName)
	assert.Equal(t, "set.sz.1", targetService.SetName)
	assert.Equal(t, "", targetService.Target)
}
