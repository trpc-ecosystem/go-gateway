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

package fhttp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	ghttp "trpc.group/trpc-go/trpc-gateway/common/http"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	thttp "trpc.group/trpc-go/trpc-go/http"
)

func TestServerCodec_Encode(t *testing.T) {
	c := DefaultServerCodec
	ctx := trpc.BackgroundContext()

	msg := codec.Message(ctx)
	_, err := c.Encode(msg, nil)
	assert.Nil(t, err)
	fctx := &fasthttp.RequestCtx{}
	fctx.Request.Header.Set(fasthttp.HeaderXForwardedFor, "127.0.0.1")
	ctx = ghttp.WithRequestContext(ctx, fctx)
	_, msg = codec.WithNewMessage(ctx)
	_, err = c.Encode(msg, []byte("ok"))
	assert.Nil(t, err)

	_, err = c.Encode(msg, nil)
	assert.Nil(t, err)

	ctx = ghttp.WithRequestContext(ctx, &fasthttp.RequestCtx{})
	_, msg = codec.WithNewMessage(ctx)
	msg.WithServerRspErr(errs.New(102, "err"))
	_, err = c.Encode(msg, nil)
	assert.Nil(t, err)
}

func TestServerCodec_Decode(t *testing.T) {
	c := DefaultServerCodec
	ctx := trpc.BackgroundContext()

	_, msg := codec.WithNewMessage(ctx)
	_, err := c.Decode(msg, nil)
	assert.NotNil(t, err)

	fctx := &fasthttp.RequestCtx{}
	fctx.URI().SetPath("/")
	fctx.Request.Header.Set(thttp.TrpcVersion, "0.8.4")
	fctx.Request.Header.Set(thttp.TrpcCallType, "6")
	fctx.Request.Header.Set(thttp.TrpcMessageType, "6")
	fctx.Request.Header.Set(thttp.TrpcRequestID, "66")
	fctx.Request.Header.Set(thttp.TrpcTimeout, "500")
	fctx.Request.Header.Set(thttp.TrpcCaller, "call")
	fctx.Request.Header.Set(thttp.TrpcCallee, "call")

	m := map[string]string{
		thttp.TrpcEnv: base64.StdEncoding.EncodeToString([]byte("development")),
	}
	dat, _ := json.Marshal(m)
	fctx.Request.Header.Set(thttp.TrpcTransInfo, string(dat))
	ctx = ghttp.WithRequestContext(ctx, fctx)
	_, msg = codec.WithNewMessage(ctx)

	_, err = c.Decode(msg, nil)
	assert.Nil(t, err)

	fctx.Request.Header.Set(thttp.TrpcTransInfo, "[]")
	ctx = ghttp.WithRequestContext(ctx, fctx)
	_, msg = codec.WithNewMessage(ctx)

	_, err = c.Decode(msg, nil)
	assert.NotNil(t, err)
}

func TestClientCodec_Decode(t *testing.T) {
	ctx := trpc.BackgroundContext()
	c := DefaultClientCodec

	_, msg := codec.WithNewMessage(ctx)
	_, err := c.Encode(msg, nil)
	assert.Nil(t, err)

	_, err = c.Decode(msg, nil)
	assert.Nil(t, err)
}

func Test_defaultErrHandler(t *testing.T) {
	defaultErrHandler(context.Background(), nil)
	fctx := &fasthttp.RequestCtx{}
	ctx := ghttp.WithRequestContext(context.Background(), fctx)
	defaultErrHandler(ctx, &errs.Error{
		Type: errs.ErrorTypeFramework,
	})

	defaultErrHandler(ctx, &errs.Error{
		Type: errs.ErrorTypeBusiness,
	})
	fctx.Response.SetStatusCode(500)
	defaultErrHandler(ctx, &errs.Error{
		Type: errs.ErrorTypeBusiness,
	})
}
