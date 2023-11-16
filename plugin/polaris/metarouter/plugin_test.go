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

package metarouter_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/plugin"
	"trpc.group/trpc-go/trpc-gateway/plugin/polaris/metarouter"
	"trpc.group/trpc-go/trpc-go/client"
	tplugin "trpc.group/trpc-go/trpc-go/plugin"
)

func TestPlugin_CheckConfig(t *testing.T) {
	p := &metarouter.Plugin{}
	tDecoder := &tplugin.YamlNodeDecoder{
		Node: &yaml.Node{},
	}
	err := p.Setup("", tDecoder)
	assert.Nil(t, err)
	opts := &metarouter.Options{
		MetaKeys: []string{"suid", ""},
	}

	// Configuration validation failed
	decoder := &plugin.PropsDecoder{Props: opts}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)
	// Configuration validation failed
	opts.MetaKeys = []string{}
	decoder = &plugin.PropsDecoder{Props: opts}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)

	// Configuration validation succeeded
	opts.MetaKeys = []string{"suid", "other"}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)

	// Plugin execution succeeded
	envHandler := metarouter.EnvHandler{}
	_, err = envHandler.ServerFilter(context.Background(), nil, func(ctx context.Context,
		req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Forward baseline environment
	fctx := &fasthttp.RequestCtx{}
	ctx := http.WithRequestContext(context.Background(), fctx)
	ctx, msg := gwmsg.WithNewGWMessage(ctx)
	msg.WithTargetService(&client.BackendConfig{
		ServiceName:          "xxxx",
		Namespace:            "Production",
		EnvName:              "formal",
		SetName:              "set.tj.0",
		Target:               "polaris://xxx",
		DisableServiceRouter: true,
	})

	fctx.Request.Header.Set("suid", "xxx")
	msg.WithPluginConfig("metarouter", nil)
	_, err = envHandler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	msg.WithPluginConfig("metarouter", "nil")
	_, err = envHandler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	msg.WithPluginConfig("metarouter", opts)
	_, err = envHandler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	msg.WithTargetService(&client.BackendConfig{
		ServiceName:          "xxxx",
		Namespace:            "Production",
		EnvName:              "formal",
		SetName:              "set.tj.0",
		Target:               "polaris://xxx",
		DisableServiceRouter: false,
	})

	msg.WithPluginConfig("metarouter", opts)
	_, err = envHandler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
}
