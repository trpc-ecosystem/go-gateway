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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/http"
)

func TestGetString(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	ret := http.GetString(ctx, "devid")
	assert.Equal(t, "", ret)
}

func TestGetParam(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	ret, ok := http.GetParam(ctx, "devid")
	assert.Equal(t, "", ret)
	assert.Equal(t, false, ok)

	ctx.PostArgs().Set("devid", "devid")
	ret, ok = http.GetParam(ctx, "devid")
	assert.Equal(t, "devid", ret)
	assert.Equal(t, true, ok)

	ctx.QueryArgs().Set("suid", "suid")
	ret, ok = http.GetParam(ctx, "suid")
	assert.Equal(t, "suid", ret)
	assert.Equal(t, true, ok)
}

func Test_getClientIPFromContext(t *testing.T) {
	localAddress := "127.0.0.1"
	fCtx := &fasthttp.RequestCtx{}
	clientIP := http.GetClientIP(fCtx)
	assert.Equal(t, "", clientIP)
	fCtx.Request.Header.Set("X-Forwarded-For", "1.1.1.1")
	clientIP = http.GetClientIP(fCtx)
	assert.Equal(t, "1.1.1.1", clientIP)

	fCtx.Request.Header.Set("X-Forwarded-For", localAddress)
	clientIP = http.GetClientIP(fCtx)
	assert.Equal(t, "", clientIP)

	fCtx.Request.Header.Set("X-Real-Ip", "1.1.1.1")
	fCtx.Request.Header.Set("X-Forwarded-For", localAddress)
	clientIP = http.GetClientIP(fCtx)
	assert.Equal(t, "1.1.1.1", clientIP)

	fCtx.Request.Header.Set("X-Real-Ip", localAddress)
	fCtx.Request.Header.Set("X-Forwarded-For", localAddress)
	clientIP = http.GetClientIP(fCtx)
	assert.Equal(t, "", clientIP)
}
