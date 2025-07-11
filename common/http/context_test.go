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

package http_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	trpc "trpc.group/trpc-go/trpc-go"
)

func TestWithRequestContext(t *testing.T) {
	ctx := trpc.BackgroundContext()
	fctx := &fasthttp.RequestCtx{}

	ctx = http.WithRequestContext(ctx, fctx)
	assert.NotNil(t, ctx)

	tmpCtx := http.RequestContext(ctx)
	assert.Equal(t, tmpCtx, fctx)

	ctx = context.WithValue(ctx, http.ContextKeyReq, "a")
	fctx = http.RequestContext(ctx)
	assert.Nil(t, fctx)
}
