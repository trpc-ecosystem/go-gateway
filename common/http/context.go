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

package http

import (
	"context"

	"github.com/valyala/fasthttp"
)

// ContextKey defines context key of http.
type ContextKey string

// ContextKeyReq key of fasthttp header
const ContextKeyReq = ContextKey("TRPC_SERVER_FASTHTTP_REQ")

// RequestContext gets the corresponding fasthttp header from context.
func RequestContext(ctx context.Context) *fasthttp.RequestCtx {
	if ret, ok := ctx.Value(ContextKeyReq).(*fasthttp.RequestCtx); ok {
		return ret
	}
	return nil
}

// WithRequestContext sets fasthttp header in context.
func WithRequestContext(ctx context.Context, val *fasthttp.RequestCtx) context.Context {
	return context.WithValue(ctx, ContextKeyReq, val)
}
