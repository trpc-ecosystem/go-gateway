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

package trpcerr2body_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/plugin"
	"trpc.group/trpc-go/trpc-gateway/plugin/transformer/trpcerr2body"
	"trpc.group/trpc-go/trpc-go/errs"
	thttp "trpc.group/trpc-go/trpc-go/http"
)

func TestPlugin_CheckConfig(t *testing.T) {
	p := &trpcerr2body.Plugin{}
	_ = p.Setup("", nil)
	p = &trpcerr2body.Plugin{}
	_ = p.Setup("", nil)
	// Validation successful
	decoder := &plugin.PropsDecoder{}
	err := p.CheckConfig("", decoder)
	assert.Nil(t, err)

	// Non-HTTP request
	_, err = trpcerr2body.ServerFilter(context.Background(), nil, func(ctx context.Context,
		req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Plugin configuration not found
	fctx := &fasthttp.RequestCtx{}
	ctx := http.WithRequestContext(context.Background(), fctx)
	ctx, msg := gwmsg.WithNewGWMessage(ctx)
	_, err = trpcerr2body.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{},
		err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Plugin configuration not found
	msg.WithPluginConfig("trpcerr2body", struct{}{})
	_, err = trpcerr2body.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{},
		err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	// Response successful
	msg.WithPluginConfig("trpcerr2body", decoder.DecodedProps)
	_, err = trpcerr2body.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{},
		err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	fctx.Response.Header.Set(thttp.TrpcFrameworkErrorCode, "")
	fctx.Response.Header.Set(thttp.TrpcUserFuncErrorCode, "400")
	fctx.Response.Header.Set(thttp.TrpcErrorMessage, "upstream err")
	// Response successful
	msg.WithPluginConfig("trpcerr2body", decoder.DecodedProps)
	_, err = trpcerr2body.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{},
		err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, `{"code":400,"msg":"upstream err"}`, string(fctx.Response.Body()))

	// Validation successful
	fctx.Response.SetBody([]byte(""))
	fctx.Response.Header.Set(thttp.TrpcFrameworkErrorCode, "400")
	decoder = &plugin.PropsDecoder{Props: &trpcerr2body.Options{
		CodePath:    "common.code",
		CodeValType: "string",
		MsgPath:     "common.msg",
	}}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig("trpcerr2body", decoder.DecodedProps)
	_, err = trpcerr2body.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{},
		err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, `{"common":{"code":"400","msg":"upstream err"}}`, string(fctx.Response.Body()))

	// Type conversion error
	fctx.Response.Header.Set(thttp.TrpcFrameworkErrorCode, "")
	fctx.Response.SetBody([]byte(""))
	decoder = &plugin.PropsDecoder{Props: &trpcerr2body.Options{
		CodePath:    "common.code",
		CodeValType: "number",
		MsgPath:     "common.msg",
	}}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig("trpcerr2body", decoder.DecodedProps)
	fctx.Response.Header.Set(thttp.TrpcUserFuncErrorCode, "xxxx")
	_, _ = trpcerr2body.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{},
		err error) {
		return nil, nil
	})
	assert.Equal(t, "", string(fctx.Response.Body()))

	// Validation framework error
	fctx.Response.SetBody([]byte(""))
	fctx.Response.Header.Set(thttp.TrpcFrameworkErrorCode, "")
	fctx.Response.Header.Set(thttp.TrpcUserFuncErrorCode, "")
	decoder = &plugin.PropsDecoder{Props: &trpcerr2body.Options{
		CodePath:    "common.code",
		CodeValType: "number",
		MsgPath:     "common.msg",
	}}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig("trpcerr2body", decoder.DecodedProps)
	_, _ = trpcerr2body.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{},
		err error) {
		return nil, errs.New(101, "request upstream err")
	})
	assert.Equal(t, `{"common":{"code":101,"msg":"request upstream err"}}`, string(fctx.Response.Body()))

	// Validation framework error
	fctx.Response.SetBody([]byte(""))
	fctx.Response.Header.Set(thttp.TrpcFrameworkErrorCode, "")
	fctx.Response.Header.Set(thttp.TrpcUserFuncErrorCode, "")
	decoder = &plugin.PropsDecoder{Props: &trpcerr2body.Options{
		CodePath:    "common.code",
		CodeValType: "number",
		MsgPath:     "common.msg",
	}}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig("trpcerr2body", decoder.DecodedProps)
	_, _ = trpcerr2body.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{},
		err error) {
		return nil, nil
	})
	assert.Equal(t, ``, string(fctx.Response.Body()))
}
