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

package logreplay_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prashantv/gostub"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/core/service/fhttp/mock"
	"trpc.group/trpc-go/trpc-gateway/plugin"
	"trpc.group/trpc-go/trpc-gateway/plugin/logreplay"
	trpc "trpc.group/trpc-go/trpc-go"
)

func TestPlugin_CheckConfig(t *testing.T) {
	p := &logreplay.Plugin{}
	_ = p.Setup("", nil)
	opts := &logreplay.Options{
		Scale: 101,
	}

	// Configuration validation fails
	decoder := &plugin.PropsDecoder{Props: opts}
	err := p.CheckConfig("", decoder)
	assert.NotNil(t, err)

	// Configuration validation succeeds
	opts.Scale = 100
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)

	lr, err := logreplay.New()
	assert.Nil(t, err)
	assert.NotNil(t, lr)
	// Failed to get plugin configuration
	_, err = lr.ServerFilter(context.Background(), nil, func(ctx context.Context,
		req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Plugin configuration assertion fails
	fctx := &fasthttp.RequestCtx{}
	ctx := http.WithRequestContext(context.Background(), fctx)
	ctx, msg := gwmsg.WithNewGWMessage(ctx)
	msg.WithPluginConfig("logreplay", struct{}{})
	_, err = lr.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Non-HTTP request
	ctx, msg = gwmsg.WithNewGWMessage(context.Background())
	msg.WithPluginConfig("logreplay", opts)
	_, err = lr.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Interface processing fails
	fctx = &fasthttp.RequestCtx{}
	ctx = http.WithRequestContext(context.Background(), fctx)
	ctx, msg = gwmsg.WithNewGWMessage(ctx)
	msg.WithPluginConfig("logreplay", opts)
	_, err = lr.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, errors.New("err")
	})
	assert.NotNil(t, err)

	// Replay succeeds
	fctx = &fasthttp.RequestCtx{}
	ctx = http.WithRequestContext(context.Background(), fctx)
	ctx, msg = gwmsg.WithNewGWMessage(ctx)
	msg.WithPluginConfig("logreplay", opts)
	_, err = lr.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Configuration validation succeeds
	opts.Scale = 0
	msg.WithPluginConfig("logreplay", opts)
	_, err = lr.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
}

func Test_replay(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	stub := gostub.New()
	defer stub.Reset()

	mockPool := mock.NewMockPool(ctrl)
	lr := &logreplay.LogReplay{
		ConnPool: mockPool,
	}

	c := &fasthttp.HostClient{
		Addr: "qq.com",
		// Proxy only makes one request, no retries allowed
		MaxIdemponentCallAttempts: 1,
	}
	mockPool.EXPECT().Get(gomock.Any()).Return(c, nil)
	mockPool.EXPECT().Put(gomock.Any()).Return(nil)

	req := &fasthttp.Request{}
	req.URI().SetPath("/user/info")
	req.URI().SetQueryString("a=1&b=2")
	req.Header.Set("header", "val")
	req.SetBodyString("body")
	trpc.GlobalConfig().Server.Service = []*trpc.ServiceConfig{
		{
			IP:   "127.0.0.1",
			Port: 8888,
		},
	}
	opts := &logreplay.Options{}
	opts.PassThroughResponse = true
	err := lr.Replay(context.Background(), req, opts)
	assert.NotNil(t, err)

	mockPool.EXPECT().Get(gomock.Any()).Return(c, errors.New("err"))
	err = lr.Replay(context.Background(), req, opts)
	assert.NotNil(t, err)

	trpc.GlobalConfig().Server.Service = []*trpc.ServiceConfig{}
	err = lr.Replay(context.Background(), req, opts)
	assert.NotNil(t, err)
}

func TestCopyRequest(t *testing.T) {
	fctx := &fasthttp.RequestCtx{}
	fctx.Request.URI().SetPath("/user/info")
	fctx.Request.URI().SetQueryString("a=1&b=2")
	fctx.Request.Header.Set("header", "val")
	fctx.Request.SetBodyString("body")
	fctx.Response.SetBodyString(`{"code":0}`)
	trpc.GlobalConfig().Server.Service = []*trpc.ServiceConfig{
		{
			IP:   "127.0.0.1",
			Port: 8080,
		},
	}
	opts := &logreplay.Options{}
	opts.PassThroughResponse = true
	lr, err := logreplay.New()
	assert.Nil(t, err)
	newReq, err := lr.CopyRequest(context.Background(), fctx, opts)
	assert.Nil(t, err)
	assert.Equal(t, "/user/info_replay", string(newReq.URI().Path()))
	assert.Equal(t, "a=1&b=2&origin_rsp_body=%257B%2522code%2522%253A0%257D", newReq.URI().QueryArgs().String())
	assert.Equal(t, "val", string(newReq.Header.Peek("header")))
	assert.Equal(t, "body", string(newReq.Body()))

	fctx.Request.Header.SetMethod(fasthttp.MethodPost)
	fctx.Request.Header.SetContentType("application/json")
	newReq, err = lr.CopyRequest(context.Background(), fctx, opts)
	assert.Nil(t, err)

	assert.Equal(t, "/user/info_replay", string(newReq.URI().Path()))
	assert.Equal(t, "a=1&b=2", newReq.URI().QueryArgs().String())
	assert.Equal(t, "val", string(newReq.Header.Peek("header")))
	assert.Equal(t, `{"origin_rsp_body":"{\"code\":0}"}`, string(newReq.Body()))

	fctx.Request.Header.SetContentType("application/x-www-form-urlencoded")
	fctx.Request.Header.SetMethod(fasthttp.MethodPost)
	newReq, err = lr.CopyRequest(context.Background(), fctx, opts)
	assert.Nil(t, err)
	assert.Equal(t, "/user/info_replay", string(newReq.URI().Path()))
	assert.Equal(t, "a=1&b=2", newReq.URI().QueryArgs().String())
	assert.Equal(t, "val", string(newReq.Header.Peek("header")))
	assert.Equal(t, `body&origin_rsp_body=%257B%2522code%2522%253A0%257D`, string(newReq.Body()))

	fctx.Request.Header.SetContentType("text/plain")
	newReq, err = lr.CopyRequest(context.Background(), fctx, opts)
	assert.Nil(t, err)
	assert.Equal(t, "/user/info_replay", string(newReq.URI().Path()))
	assert.Equal(t, "a=1&b=2&origin_rsp_body=%257B%2522code%2522%253A0%257D", newReq.URI().QueryArgs().String())
	assert.Equal(t, "val", string(newReq.Header.Peek("header")))
	assert.Equal(t, `body`, string(newReq.Body()))

	fctx.Request.Header.SetContentType(`"foo; filename=bar;baz"; filename=qux`)
	_, err = lr.CopyRequest(context.Background(), fctx, opts)
	assert.NotNil(t, err)
}
