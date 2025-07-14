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

package fasthttp

import (
	"context"

	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/core/service/protocol"
	"trpc.group/trpc-go/trpc-gateway/internal"
	"trpc.group/trpc-go/trpc-go/client"
)

func init() {
	protocol.RegisterCliProtocolHandler(Protocol, &defaultProtocolHandler{})
}

// Protocol protocol name
const Protocol = "fasthttp"

// defaultProtocolHandler default protocol handler
type defaultProtocolHandler struct{}

// WithCtx set context
func (h *defaultProtocolHandler) WithCtx(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

// GetCliOptions get specific client options for the request
func (h *defaultProtocolHandler) GetCliOptions(context.Context) ([]client.Option, error) {
	return nil, nil
}

// TransReqBody transform request body
func (h *defaultProtocolHandler) TransReqBody(ctx context.Context) (interface{}, error) {
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return nil, nil
	}
	// Force conversion to HTTP request
	fctx.Request.URI().SetScheme("http")
	// Remove hop-by-hop request headers
	for _, h := range internal.HopHeaders {
		fctx.Request.Header.Del(h)
	}
	return nil, nil
}

// TransRspBody transform response body
func (h *defaultProtocolHandler) TransRspBody(context.Context) (interface{}, error) {
	return nil, nil
}

// HandleErr handle error information
func (h *defaultProtocolHandler) HandleErr(ctx context.Context, err error) error {
	if err == nil {
		return err
	}
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return err
	}
	if fctx.Response.StatusCode() == fasthttp.StatusOK {
		fctx.Response.SetStatusCode(fasthttp.StatusInternalServerError)
	}
	return err
}

// HandleRspBody handle response
func (h *defaultProtocolHandler) HandleRspBody(ctx context.Context, _ interface{}) error {
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return nil
	}
	// Remove hop-by-hop response headers
	for _, h := range internal.HopHeaders {
		fctx.Response.Header.Del(h)
	}
	return nil
}
