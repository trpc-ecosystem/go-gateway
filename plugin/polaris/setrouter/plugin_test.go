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

package setrouter_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/plugin"
	"trpc.group/trpc-go/trpc-gateway/plugin/polaris/setrouter"
	tplugin "trpc.group/trpc-go/trpc-go/plugin"
)

func TestPlugin_CheckConfig(t *testing.T) {
	p := &setrouter.Plugin{}
	tDecoder := &tplugin.YamlNodeDecoder{
		Node: &yaml.Node{},
	}
	err := p.Setup("", tDecoder)
	assert.Nil(t, err)
	opts := &setrouter.Options{}

	// Configuration validation fails
	decoder := &plugin.PropsDecoder{Props: opts}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)
	// Configuration validation fails
	opts.SetName = ""
	decoder = &plugin.PropsDecoder{Props: opts}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)

	// Configuration validation succeeds
	opts.SetName = "set.tj.1"
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)

	// Plugin execution succeeds
	handler := setrouter.SetHandler{}
	_, err = handler.ServerFilter(context.Background(), nil, func(ctx context.Context,
		req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Forwarding baseline environment
	fctx := &fasthttp.RequestCtx{}
	ctx := http.WithRequestContext(context.Background(), fctx)
	ctx, msg := gwmsg.WithNewGWMessage(ctx)

	fctx.Request.Header.Set("suid", "xxx")
	msg.WithPluginConfig("setrouter", nil)
	_, err = handler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.NotNil(t, err)

	msg.WithPluginConfig("setrouter", "nil")
	_, err = handler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.NotNil(t, err)

	msg.WithPluginConfig("setrouter", decoder.DecodedProps)
	_, err = handler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	msg.WithPluginConfig("setrouter", decoder.DecodedProps)
	_, err = handler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	opts.SetName = "set.tj.1"
	msg.WithPluginConfig("setrouter", opts)
	_, err = handler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	opts.SetName = "set.tj.1"
	msg.WithPluginConfig("setrouter", opts)
	_, err = handler.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
}
