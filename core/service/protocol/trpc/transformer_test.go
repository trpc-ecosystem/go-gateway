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

package trpc_test

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/core/service/protocol/trpc"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	thttp "trpc.group/trpc-go/trpc-go/http"
)

func Test_trpcProtocolHandler_HandleErr(t *testing.T) {
	h := trpc.DefaultTRPCProtocolHandler
	err := h.HandleErr(context.Background(), nil)
	assert.Nil(t, err)
	err = h.HandleErr(context.Background(), errs.New(1, "business err"))
	assert.NotNil(t, err)

	fctx := &fasthttp.RequestCtx{}
	ctx := http.WithRequestContext(context.Background(), fctx)
	err = h.HandleErr(ctx, errors.New("err"))
	assert.NotNil(t, err)
	err = errs.NewFrameError(1, "frame err")
	err = h.HandleErr(ctx, err)
	assert.NotNil(t, err)

	err = errs.New(2, "business err")
	err = h.HandleErr(ctx, err)
	assert.Nil(t, err)
	assert.Equal(t, "business err", string(fctx.Response.Header.Peek(thttp.TrpcErrorMessage)))
	assert.Equal(t, "2", string(fctx.Response.Header.Peek(thttp.TrpcUserFuncErrorCode)))
}

func Test_trpcProtocolHandler_TransReqBody(t *testing.T) {

	h := trpc.DefaultTRPCProtocolHandler
	_, err := h.TransReqBody(context.Background())
	assert.NotNil(t, err)

	fctx := &fasthttp.RequestCtx{}
	ctx := http.WithRequestContext(context.Background(), fctx)
	_, err = h.TransReqBody(ctx)
	assert.Nil(t, err)

	// empty content-type
	fctx.Request.Header.SetMethod(fasthttp.MethodPost)
	_, err = h.TransReqBody(ctx)
	assert.Nil(t, err)

	fctx.Request.Header.SetContentType("application/json")
	_, err = h.TransReqBody(ctx)
	assert.Nil(t, err)
	fctx.Request.Header.SetContentType("application/x-www-form-urlencoded")
	_, err = h.TransReqBody(ctx)
	assert.Nil(t, err)

	fctx.Request.Header.SetContentType("application/protobuf")
	_, err = h.TransReqBody(ctx)
	assert.Nil(t, err)

	fctx.Request.Header.SetContentType("application/proto")
	_, err = h.TransReqBody(ctx)
	assert.Nil(t, err)

	fctx.Request.Header.SetContentType("multipart/form-data")
	_, err = h.TransReqBody(ctx)
	assert.NotNil(t, err)

	s := "POST / HTTP/1.1\r\nHost: aaa\r\nContent-Type: multipart/form-data; boundary=foobar\r\nContent-Length: 213\r\n\r\n--foobar\r\nContent-Disposition: form-data; name=\"key_0\"\r\n\r\nvalue_0\r\n--foobar\r\nContent-Disposition: form-data; name=\"key_1\"\r\n\r\nvalue_1\r\n--foobar\r\nContent-Disposition: form-data; name=\"key_2\"\r\n\r\nvalue_2\r\n--foobar--\r\n"
	r := bytes.NewBufferString(s)
	br := bufio.NewReader(r)
	if err := fctx.Request.Read(br); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fctx.Request.Header.SetContentType("multipart/form-data")
	_, err = h.TransReqBody(ctx)
	assert.Nil(t, err)

	fctx.Request.Header.SetContentType("application/invalid")
	_, err = h.TransReqBody(ctx)
	assert.NotNil(t, err)

}

func Test_trpcProtocolHandler_TransRspBody(t *testing.T) {
	h := trpc.DefaultTRPCProtocolHandler
	got, err := h.TransRspBody(nil)
	assert.Nil(t, err)
	assert.NotNil(t, got)
}

func Test_trpcProtocolHandler_GetCliOptions(t *testing.T) {
	h := trpc.DefaultTRPCProtocolHandler
	_, err := h.GetCliOptions(context.Background())
	assert.NotNil(t, err)
	fctx := &fasthttp.RequestCtx{}
	ctx := http.WithRequestContext(context.Background(), fctx)
	_, err = h.GetCliOptions(ctx)
	assert.Nil(t, err)

	fctx.Request.Header.SetContentType("application/json")
	fctx.Request.URI().QueryArgs().Set("a", "b")
	fctx.Request.URI().SetQueryString(fctx.Request.URI().QueryArgs().String())
	got, err := h.GetCliOptions(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(got))
	o := &client.Options{}
	for _, opt := range got {
		opt(o)
	}
	header, _ := url.ParseQuery(string(o.MetaData[http.TRPCGatewayHTTPHeader]))
	assert.Equal(t, "application/json", header.Get("Content-Type"))
	query, _ := url.ParseQuery(string(o.MetaData[http.TRPCGatewayHTTPQuery]))
	assert.Equal(t, "b", query.Get("a"))
}

func Test_trpcProtocolHandler_WithCtx(t *testing.T) {
	h := trpc.DefaultTRPCProtocolHandler
	got, err := h.WithCtx(context.Background())
	assert.Nil(t, err)
	assert.NotNil(t, got)
}

func Test_trpcProtocolHandler_HandleRspBody(t *testing.T) {
	h := trpc.DefaultTRPCProtocolHandler
	err := h.HandleRspBody(context.Background(), nil)
	assert.Nil(t, err)

	ctx, msg := codec.WithNewMessage(context.Background())
	msg.WithClientRspHead("xxx")
	fctx := &fasthttp.RequestCtx{}
	ctx = http.WithRequestContext(ctx, fctx)
	err = h.HandleRspBody(ctx, nil)
	assert.Nil(t, err)
	rspBody := &codec.Body{
		Data: []byte("xxx"),
	}
	err = h.HandleRspBody(ctx, rspBody)
	assert.Nil(t, err)

	fctx.Request.Header.SetMethod("POST")
	fctx.Request.Header.SetContentType("application/json")
	err = h.HandleRspBody(ctx, rspBody)
	assert.Nil(t, err)

	fctx.Request.Header.SetContentType("multipart/form-data")
	err = h.HandleRspBody(ctx, rspBody)
	assert.Nil(t, err)
}
