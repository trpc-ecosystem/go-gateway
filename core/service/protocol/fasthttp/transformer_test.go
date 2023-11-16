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

package fasthttp

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-go/errs"
)

func Test_defaultProtocolHandler_HandleErr(t *testing.T) {
	h := &defaultProtocolHandler{}
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
	assert.NotNil(t, err)
}

func Test_defaultProtocolHandler_TransReqBody(t *testing.T) {
	h := &defaultProtocolHandler{}
	got, err := h.TransReqBody(context.Background())
	assert.Nil(t, err)
	assert.Nil(t, got)
	err = h.HandleRspBody(context.Background(), nil)
	assert.Nil(t, err)
	fctx := &fasthttp.RequestCtx{}
	ctx := http.WithRequestContext(context.Background(), fctx)
	got, err = h.TransReqBody(ctx)
	assert.Nil(t, err)
	assert.Nil(t, got)
	rb, err := h.TransRspBody(ctx)
	assert.Nil(t, rb)
	assert.Nil(t, err)

	opts, err := h.GetCliOptions(ctx)
	assert.Nil(t, err)
	assert.Nil(t, opts)

	ctx, _ = h.WithCtx(ctx)
	assert.NotNil(t, ctx)

	err = h.HandleRspBody(ctx, nil)
	assert.Nil(t, err)
}
