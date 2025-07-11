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

package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/http"
)

func Test_gRPCProtocolHandler_WithCtx(t *testing.T) {
	h := &gRPCProtocolHandler{}
	got, err := h.WithCtx(context.Background())
	assert.Nil(t, err)
	assert.NotNil(t, got)
	_, err = h.GetCliOptions(context.Background())
	assert.Nil(t, err)
	_, err = h.TransRspBody(context.Background())
	assert.Nil(t, err)

	err = h.HandleErr(context.Background(), errors.New("err"))
	assert.NotNil(t, err)
}

func Test_gRPCProtocolHandler_TransReqBody(t *testing.T) {

	h := &gRPCProtocolHandler{}
	_, err := h.TransReqBody(context.Background())
	assert.NotNil(t, err)

	fctx := &fasthttp.RequestCtx{}
	ctx := http.WithRequestContext(context.Background(), fctx)
	_, err = h.TransReqBody(ctx)
	assert.NotNil(t, err)

	fctx.Request.Header.Set("a", "1")
	fctx.Request.URI().QueryArgs().Set("b", "1")
	ctx = WithHeader(ctx, &Header{})
	_, err = h.TransReqBody(ctx)
	assert.Nil(t, err)
}

func Test_gRPCProtocolHandler_HandleRspBody(t *testing.T) {

	h := &gRPCProtocolHandler{}
	err := h.HandleRspBody(context.Background(), nil)
	assert.Nil(t, err)
	fctx := &fasthttp.RequestCtx{}
	ctx := http.WithRequestContext(context.Background(), fctx)

	err = h.HandleRspBody(ctx, nil)
	assert.NotNil(t, err)
	ctx = WithHeader(ctx, &Header{})
	err = h.HandleRspBody(ctx, nil)
	assert.Nil(t, err)

}

func Test_gRPCProtocolHandler_Extract(t *testing.T) {
	h := &gRPCProtocolHandler{}

	fctx := &fasthttp.RequestCtx{}
	fctx.Request.Header.Set("User-Agent", "curl/7.54.0")
	fctx.Request.Header.Set("X-Forward-For", "127.0.0.1")
	fctx.Request.Header.SetCookie("uid", "123")

	var u fasthttp.URI
	err := u.Parse([]byte("demo.com"), []byte("/users/123/foo?bar=baz&baraz#qqqq"))
	assert.Nil(t, err)

	fctx.Request.SetURI(&u)

	ctx := WithHeader(context.Background(), &Header{})
	ctx = http.WithRequestContext(ctx, fctx)

	_, err = h.TransReqBody(ctx)
	assert.Nil(t, err)

	assert.NotNil(t, Head(ctx))

	// key of grpc metadata auto convert lower case
	assert.NotEmpty(t, Head(ctx).OutMetadata["trpc_gateway_http_header"])
	assert.NotEmpty(t, Head(ctx).OutMetadata["trpc_gateway_http_query"])

	assert.Equal(t, Head(ctx).OutMetadata["trpc_gateway_http_header"][0],
		"Cookie=uid%3D123&User-Agent=curl%2F7.54.0&X-Forward-For=127.0.0.1") //
	assert.Equal(t, Head(ctx).OutMetadata["trpc_gateway_http_query"][0], "bar=baz&baraz")
}
