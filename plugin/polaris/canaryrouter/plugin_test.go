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

package canaryrouter_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/plugin"
	"trpc.group/trpc-go/trpc-gateway/plugin/polaris/canaryrouter"
	tplugin "trpc.group/trpc-go/trpc-go/plugin"
)

func TestPlugin_CheckConfig(t *testing.T) {
	p := &canaryrouter.Plugin{}
	tDecoder := &tplugin.YamlNodeDecoder{
		Node: &yaml.Node{},
	}
	err := p.Setup("", tDecoder)
	assert.Nil(t, err)
	opts := &canaryrouter.Options{
		ReqKey: "",
		Values: nil,
	}

	// Configuration validation failed
	decoder := &plugin.PropsDecoder{Props: opts}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	// Configuration validation failed
	opts.ReqKey = "suid"
	opts.Values = []string{}
	decoder = &plugin.PropsDecoder{Props: opts}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)

	// Configuration validation failed
	opts.ReqKey = "suid"
	opts.Values = []string{"xxx"}
	opts.Scale = 101
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)

	// Configuration validation succeeded
	opts.ReqKey = "suid"
	opts.Values = []string{"xxx"}
	opts.Scale = 50
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)

	// Plugin execution succeeded
	handler := canaryrouter.CanaryHandler{}
	_, err = handler.ServerFilter(context.Background(), nil, func(ctx context.Context,
		req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Forward baseline environment
	fctx := &fasthttp.RequestCtx{}
	ctx := http.WithRequestContext(context.Background(), fctx)
	ctx, msg := gwmsg.WithNewGWMessage(ctx)

	fctx.Request.Header.Set("suid", "xxx")
	msg.WithPluginConfig("metarouter", nil)
	_, err = handler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	msg.WithPluginConfig("canaryrouter", "nil")
	_, err = handler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	msg.WithPluginConfig("canaryrouter", decoder.DecodedProps)
	_, err = handler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	msg.WithPluginConfig("canaryrouter", decoder.DecodedProps)
	_, err = handler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	opts.ReqKey = "qimei36"
	msg.WithPluginConfig("canaryrouter", opts)
	_, err = handler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	opts.Scale = 100
	msg.WithPluginConfig("canaryrouter", opts)
	_, err = handler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	opts.ReqKey = ""
	opts.Scale = 99
	msg.WithPluginConfig("canaryrouter", opts)
	_, err = handler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	opts.Scale = 99
	opts.HashKey = "suid"
	msg.WithPluginConfig("canaryrouter", opts)
	_, err = handler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	opts.Scale = 99
	opts.HashKey = "empty"
	msg.WithPluginConfig("canaryrouter", opts)
	_, err = handler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	opts.ReqKey = "client_ip"
	msg.WithPluginConfig("canaryrouter", opts)
	_, err = handler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
}
