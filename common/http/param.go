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

// Package http contains utility methods related to HTTP.
package http

import (
	"strings"

	"github.com/valyala/fasthttp"
)

// GetString returns HTTP request parameters.
func GetString(fctx *fasthttp.RequestCtx, key string) (ret string) {
	ret = string(fctx.QueryArgs().Peek(key))

	if ret == "" {
		ret = string(fctx.PostArgs().Peek(key))
	}

	if "" == ret {
		ret = string(fctx.Request.Header.Peek(key))
	}

	if "" == ret {
		ret = string(fctx.Request.Header.Cookie(key))
	}

	if "" == ret {
		ret = string(fctx.Request.Header.Cookie("%20" + key))
	}
	return
}

// GetParam retrieves the parameter and checks if it exists or not.
func GetParam(ctx *fasthttp.RequestCtx, key string) (string, bool) {
	if ctx.PostArgs().Has(key) {
		return string(ctx.PostArgs().Peek(key)), true
	}

	if ctx.QueryArgs().Has(key) {
		return string(ctx.QueryArgs().Peek(key)), true
	}

	return "", false
}

const localAddress = "127.0.0.1"

// GetClientIP returns the client's IP address.
func GetClientIP(fctx *fasthttp.RequestCtx) string {
	clientIPByte := fctx.Request.Header.Peek(fasthttp.HeaderXForwardedFor)
	clientIPs := strings.Split(string(clientIPByte), ",")
	clientIP := clientIPs[0]
	if len(clientIP) > 0 && clientIP != localAddress {
		return clientIP
	}
	clientIP = strings.TrimSpace(string(fctx.Request.Header.Peek("X-Real-Ip")))
	if clientIP != "" && clientIP != localAddress {
		return clientIP
	}
	return ""
}
