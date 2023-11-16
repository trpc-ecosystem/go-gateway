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

package grpc

import (
	"context"
	"net/http"

	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	thttp "trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/core/service/protocol"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
)

type gRPCProtocolHandler struct{}

func init() {
	protocol.RegisterCliProtocolHandler(Protocol, &gRPCProtocolHandler{})
}

// WithCtx sets the context
func (h *gRPCProtocolHandler) WithCtx(ctx context.Context) (context.Context, error) {
	return WithHeader(ctx, &Header{}), nil
}

// GetCliOptions retrieves specific client options for the request
func (h *gRPCProtocolHandler) GetCliOptions(_ context.Context) ([]client.Option, error) {
	return nil, nil
}

// TransReqBody converts the request body
func (h *gRPCProtocolHandler) TransReqBody(ctx context.Context) (interface{}, error) {
	fctx := thttp.RequestContext(ctx)
	if fctx == nil {
		return nil, errs.New(gerrs.ErrWrongContext, "not http request")
	}
	header := Head(ctx)
	if header == nil {
		return nil, errs.New(gerrs.ErrUnSupportProtocol, "get no grpc header get req body")
	}
	header.Req = fctx.Request.Body()

	// Put all request headers into metadata
	stdh := make(http.Header)
	fctx.Request.Header.VisitAll(func(key, value []byte) {
		stdh.Add(string(key), string(value))
	})
	WithServerGRPCMetadata(ctx, thttp.TRPCGatewayHTTPHeader, []string{thttp.EncodeHTTPHeaders(stdh)})
	// Put query parameters into metadata
	WithServerGRPCMetadata(ctx, thttp.TRPCGatewayHTTPQuery, []string{string(fctx.Request.URI().QueryString())})

	// Set serialization type, no need to handle
	codec.Message(ctx).WithSerializationType(codec.SerializationTypeUnsupported)
	return nil, nil
}

// TransRspBody converts the response body
func (h *gRPCProtocolHandler) TransRspBody(_ context.Context) (interface{}, error) {
	return nil, nil
}

// HandleErr handles error messages
func (h *gRPCProtocolHandler) HandleErr(_ context.Context, err error) error {
	return err
}

// HandleRspBody handles the response body
func (h *gRPCProtocolHandler) HandleRspBody(ctx context.Context, _ interface{}) error {
	fctx := thttp.RequestContext(ctx)
	if fctx == nil {
		return nil
	}

	fctx.Response.Header.SetContentType("application/json")
	header := Head(ctx)
	if header == nil {
		return errs.New(gerrs.ErrGatewayUnknown, "get no grpc header")
	}
	// Set it to gwmsg for plugin usage
	gwmsg.GwMessage(ctx).WithUpstreamRspHead(header)
	_, err := fctx.Write(header.Rsp)
	if err != nil {
		return gerrs.Wrap(err, "write_rsp_body_err")
	}
	return nil
}
