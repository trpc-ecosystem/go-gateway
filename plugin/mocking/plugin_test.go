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

package mocking_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/plugin"
	"trpc.group/trpc-go/trpc-gateway/plugin/mocking"
)

func TestPlugin_CheckConfig(t *testing.T) {
	p := &mocking.Plugin{}
	_ = p.Setup("", nil)
	opts := &mocking.Options{
		Delay:           1,
		ResponseStatus:  0,
		ContentType:     "",
		ResponseExample: "",
		WithMockHeader:  true,
		Scale:           0,
	}

	// Configuration validation fails
	decoder := &plugin.PropsDecoder{Props: opts}
	err := p.CheckConfig("", decoder)
	assert.NotNil(t, err)

	mockBody := `{"code":0,"msg":"success"}`
	// Configuration validation succeeds
	opts.ResponseExample = mockBody
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)

	// Non-HTTP request
	_, err = mocking.ServerFilter(context.Background(), nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Plugin configuration not obtained
	fctx := &fasthttp.RequestCtx{}
	ctx := http.WithRequestContext(context.Background(), fctx)
	ctx, msg := gwmsg.WithNewGWMessage(ctx)
	_, err = mocking.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.NotNil(t, err)

	// Plugin configuration not obtained
	msg.WithPluginConfig("mocking", struct{}{})
	_, err = mocking.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.NotNil(t, err)

	// Successful response
	msg.WithPluginConfig("mocking", opts)
	_, err = mocking.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, mockBody, string(fctx.Response.Body()))

	opts.Scale = 99.99
	_, err = mocking.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, mockBody, string(fctx.Response.Body()))

	opts.HashKey = "suid"
	_, err = mocking.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, mockBody, string(fctx.Response.Body()))

	fctx.Request.Header.Set("suid", "xxxx")
	_, err = mocking.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, mockBody, string(fctx.Response.Body()))

	opts.HashKey = "client_ip"
	_, err = mocking.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, mockBody, string(fctx.Response.Body()))
}
