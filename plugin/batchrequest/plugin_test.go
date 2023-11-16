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

package batchrequest_test

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
	"trpc.group/trpc-go/trpc-gateway/plugin"
	"trpc.group/trpc-go/trpc-gateway/plugin/batchrequest"
	trpc "trpc.group/trpc-go/trpc-go"
)

func TestPlugin_CheckConfig(t *testing.T) {
	userInfoPath := "/user/info"
	feedInfoPath := "/feed/info"
	p := &batchrequest.Plugin{}
	_ = p.Setup("", nil)
	opts := &batchrequest.Options{
		CodePath:    "",
		MsgPath:     "",
		SuccessCode: 0,
	}
	opts.RequestList = []*batchrequest.Request{
		{},
		{},
		{},
		{},
		{},
		{},
	}

	// Configuration validation failed
	decoder := &plugin.PropsDecoder{Props: opts}
	err := p.CheckConfig("", decoder)
	assert.NotNil(t, err)

	opts.RequestList = []*batchrequest.Request{
		{
			Method:         "",
			SourceDatePath: "data",
			TargetDataPath: "user",
			IgnoreErr:      false,
			CodePath:       "",
			MsgPath:        "",
			SuccessCode:    0,
		},
		{
			Method:         feedInfoPath,
			SourceDatePath: "data",
			TargetDataPath: "feed",
			IgnoreErr:      false,
			CodePath:       "",
			MsgPath:        "",
			SuccessCode:    0,
		},
	}
	// Configuration validation failed
	decoder = &plugin.PropsDecoder{Props: opts}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)

	opts.RequestList = []*batchrequest.Request{
		{
			Method:         userInfoPath,
			SourceDatePath: "",
			TargetDataPath: "user",
			IgnoreErr:      false,
			CodePath:       "",
			MsgPath:        "",
			SuccessCode:    0,
		},
		{
			Method:         feedInfoPath,
			SourceDatePath: "data",
			TargetDataPath: "feed",
			IgnoreErr:      false,
			CodePath:       "",
			MsgPath:        "",
			SuccessCode:    0,
		},
	}

	// Configuration validation failed
	decoder = &plugin.PropsDecoder{Props: opts}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)

	opts.RequestList = []*batchrequest.Request{
		{
			Method:         userInfoPath,
			SourceDatePath: "data",
			TargetDataPath: "user",
			IgnoreErr:      false,
			CodePath:       "",
			MsgPath:        "",
			SuccessCode:    0,
		},
		{
			Method:         userInfoPath,
			SourceDatePath: "data",
			TargetDataPath: "feed",
			IgnoreErr:      false,
			CodePath:       "",
			MsgPath:        "",
			SuccessCode:    0,
		},
	}

	// Configuration validation failed
	decoder = &plugin.PropsDecoder{Props: opts}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)

	// Configuration validation successful
	opts.RequestList[1].Method = feedInfoPath
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	// Failed to retrieve plugin configuration
	_, err = batchrequest.ServerFilter(context.Background(), nil, func(ctx context.Context,
		req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Plugin configuration assertion failed
	fctx := &fasthttp.RequestCtx{}
	ctx := http.WithRequestContext(context.Background(), fctx)
	ctx, msg := gwmsg.WithNewGWMessage(ctx)
	msg.WithPluginConfig("batch_request", struct{}{})
	_, err = batchrequest.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Non-HTTP request
	ctx, msg = gwmsg.WithNewGWMessage(context.Background())
	msg.WithPluginConfig("batch_request", decoder.DecodedProps)
	_, err = batchrequest.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Send request
	trpc.GlobalConfig().Server.Service = []*trpc.ServiceConfig{{
		IP:   "127.0.0.1",
		Port: 8888,
	}}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	stub := gostub.New()
	defer stub.Reset()
	stub.Stub(&batchrequest.FasthttpDo, func(req *fasthttp.Request, resp *fasthttp.Response) error {
		if string(req.URI().Path()) == userInfoPath {
			resp.SetBody([]byte(`{"code":0,"msg":"success","data":{"user":"bbbb"}}`))
		} else {
			resp.SetBody([]byte(`{"code":0,"msg":"success","data":{"feed":"aaaa"}}`))
		}
		return nil
	})

	fctx = &fasthttp.RequestCtx{}
	ctx = http.WithRequestContext(context.Background(), fctx)
	ctx, msg = gwmsg.WithNewGWMessage(ctx)
	msg.WithPluginConfig("batch_request", decoder.DecodedProps)
	_, err = batchrequest.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, `{"code":0,"user":{"user":"bbbb"},"feed":{"feed":"aaaa"}}`, string(fctx.Response.Body()))

	// One success, one failure
	stub.Stub(&batchrequest.FasthttpDo, func(req *fasthttp.Request, resp *fasthttp.Response) error {
		if string(req.URI().Path()) == userInfoPath {
			resp.SetBody([]byte(`{"code":0,"msg":"success","data":{"user":"bbbb"}}`))
			return errors.New("err")
		} else {
			resp.SetBody([]byte(`{"code":0,"msg":"success","data":{"feed":"aaaa"}}`))
		}
		return nil
	})

	opts.RequestList[0].IgnoreErr = true
	err = p.CheckConfig("", decoder)

	msg.WithPluginConfig("batch_request", decoder.DecodedProps)
	_, err = batchrequest.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, `{"code":0,"feed":{"feed":"aaaa"}}`, string(fctx.Response.Body()))

	// One success, one failure, without ignoring errors
	opts.RequestList[0].IgnoreErr = false
	err = p.CheckConfig("", decoder)

	msg.WithPluginConfig("batch_request", decoder.DecodedProps)
	_, err = batchrequest.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.NotNil(t, err)

	// One success, one failure, ignoring errors when code != 0
	stub.Stub(&batchrequest.FasthttpDo, func(req *fasthttp.Request, resp *fasthttp.Response) error {
		if string(req.URI().Path()) == userInfoPath {
			resp.SetBody([]byte(`{"code":500,"msg":"success","data":{"user":"bbbb"}}`))
		} else {
			resp.SetBody([]byte(`{"code":0,"msg":"success","data":{"feed":"aaaa"}}`))
		}
		return nil
	})
	opts.RequestList[0].IgnoreErr = true
	err = p.CheckConfig("", decoder)

	msg.WithPluginConfig("batch_request", decoder.DecodedProps)
	_, err = batchrequest.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Equal(t, `{"code":0,"feed":{"feed":"aaaa"}}`, string(fctx.Response.Body()))

	// Do not ignore errors
	opts.RequestList[0].IgnoreErr = false
	err = p.CheckConfig("", decoder)

	msg.WithPluginConfig("batch_request", decoder.DecodedProps)
	_, err = batchrequest.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.NotNil(t, err)
}

func TestDoRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	stub := gostub.New()
	defer stub.Reset()
	stub.Stub(&batchrequest.FasthttpDo, func(req *fasthttp.Request, resp *fasthttp.Response) error {
		resp.SetBody([]byte(`{"code":0}`))
		return nil
	})
	request := &batchrequest.Request{}
	request.Method = "/tag/info"

	got, err := batchrequest.DoRequest(context.Background(), request)
	assert.Nil(t, err)

	fctx := &fasthttp.RequestCtx{}
	ctx := http.WithRequestContext(context.Background(), fctx)
	got, err = batchrequest.DoRequest(ctx, request)
	assert.Nil(t, err)
	assert.Equal(t, `{"code":0}`, string(got))

	got, err = batchrequest.DoRequest(ctx, nil)
	assert.NotNil(t, err)

	trpc.GlobalConfig().Server.Service = []*trpc.ServiceConfig{}
	got, err = batchrequest.DoRequest(ctx, request)
	assert.NotNil(t, err)

	// Send request
	trpc.GlobalConfig().Server.Service = []*trpc.ServiceConfig{{
		IP:   "127.0.0.1",
		Port: 8888,
	}}

	stub.Stub(&batchrequest.FasthttpDo, func(req *fasthttp.Request, resp *fasthttp.Response) error {
		resp.SetStatusCode(fasthttp.StatusServiceUnavailable)
		resp.SetBody([]byte(`{"code":0}`))
		return nil
	})

	_, err = batchrequest.DoRequest(ctx, request)
	assert.NotNil(t, err)
}
