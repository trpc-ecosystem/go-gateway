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

package trpc

import (
	"context"
	"errors"
	"mime"
	"net/http"
	"net/url"
	"strconv"

	"github.com/valyala/fasthttp"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	chttp "trpc.group/trpc-go/trpc-gateway/common/http"
	ctrpc "trpc.group/trpc-go/trpc-gateway/common/trpc"
	"trpc.group/trpc-go/trpc-gateway/core/service/protocol"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	thttp "trpc.group/trpc-go/trpc-go/http"
)

func init() {
	protocol.RegisterCliProtocolHandler("trpc", DefaultTRPCProtocolHandler)
}

type trpcProtocolHandler struct{}

// DefaultTRPCProtocolHandler 默认 trpc 协议转发处理器
var DefaultTRPCProtocolHandler = &trpcProtocolHandler{}

// WithCtx sets the context
func (h *trpcProtocolHandler) WithCtx(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

// GetCliOptions gets specific client options for the request
func (h *trpcProtocolHandler) GetCliOptions(ctx context.Context) ([]client.Option, error) {
	fctx := chttp.RequestContext(ctx)
	if fctx == nil {
		return nil, errs.New(gerrs.ErrWrongContext, "not an HTTP request")
	}
	var opts []client.Option

	// Put all request headers into metadata
	stdh := make(http.Header)
	fctx.Request.Header.VisitAll(func(key, value []byte) {
		stdh.Add(string(key), string(value))
	})
	opts = append(opts, client.WithMetaData(chttp.TRPCGatewayHTTPHeader, []byte(chttp.EncodeHTTPHeaders(stdh))))
	// Put query parameters into metadata
	opts = append(opts, client.WithMetaData(chttp.TRPCGatewayHTTPQuery, fctx.Request.URI().QueryString()))

	opts = append(opts, client.WithCurrentSerializationType(codec.SerializationTypeNoop))
	return opts, nil
}

// TransReqBody transforms the request body
func (h *trpcProtocolHandler) TransReqBody(ctx context.Context) (interface{}, error) {
	msg := codec.Message(ctx)
	fctx := chttp.RequestContext(ctx)
	if fctx == nil {
		return nil, errs.New(gerrs.ErrWrongContext, "not an HTTP request")
	}
	if fctx.Request.Header.IsGet() {
		msg.WithSerializationType(codec.SerializationTypeGet)
		return &codec.Body{Data: fctx.Request.URI().QueryString()}, nil
	}
	// Get the serialization type
	st, err := getSerializationType(ctx)
	if err != nil {
		return nil, gerrs.Wrap(err, "error getting request serialization type")
	}
	msg.WithSerializationType(st)
	if st == codec.SerializationTypeFormData {
		return h.getMultipartFormBody(fctx)
	}
	return &codec.Body{Data: fctx.Request.Body()}, nil
}

// Convert multipartForm parameters to query parameters
func (h *trpcProtocolHandler) getMultipartFormBody(fctx *fasthttp.RequestCtx) (interface{}, error) {
	f, err := fctx.Request.MultipartForm()
	if err != nil {
		return nil, errs.Wrap(err, gerrs.ErrProtocolTrans, "error getting multipart form")
	}
	defer fctx.Request.RemoveMultipartFormFiles()
	form := make(url.Values)
	for k, v := range f.Value {
		form[k] = append(form[k], v...)
	}
	return &codec.Body{Data: []byte(form.Encode())}, nil
}

// TransRspBody transforms the response body
func (h *trpcProtocolHandler) TransRspBody(_ context.Context) (interface{}, error) {
	return &codec.Body{}, nil
}

// HandleErr writes trpc error information to the response headers, ensuring that the calling trpc client can retrieve
// the error information
func (h *trpcProtocolHandler) HandleErr(ctx context.Context, err error) error {
	if err == nil {
		return err
	}
	fCtx := chttp.RequestContext(ctx)
	if fCtx == nil {
		return err
	}

	var te *errs.Error
	if ok := errors.As(err, &te); !ok {
		return err
	}

	if te.Type == errs.ErrorTypeBusiness {
		// Business error code, indicates successful forwarding, do not return error
		fCtx.Response.Header.Set(thttp.TrpcErrorMessage, errs.Msg(err))
		fCtx.Response.Header.Set(thttp.TrpcUserFuncErrorCode, strconv.Itoa(int(errs.Code(err))))
		return nil
	}
	return err
}

// HandleRspBody handles the response
func (h *trpcProtocolHandler) HandleRspBody(ctx context.Context, rspBody interface{}) error {
	// Store the backend service response in gwmsg for plugins to access metadata, etc.
	gwmsg.GwMessage(ctx).WithUpstreamRspHead(codec.Message(ctx).ClientRspHead())

	fctx := chttp.RequestContext(ctx)
	if fctx == nil {
		return nil
	}
	contentType, err := getContentType(fctx)
	if err != nil {
		return gerrs.Wrap(err, "error getting response content type")
	}

	fctx.Response.Header.SetContentType(contentType)
	if rspBody == nil {
		return nil
	}
	if _, err := fctx.Write(rspBody.(*codec.Body).Data); err != nil {
		return gerrs.Wrap(err, "error writing response body")
	}
	return nil
}

func getContentType(fctx *fasthttp.RequestCtx) (string, error) {
	defaultContentType := "application/json"
	if fctx.IsGet() || string(fctx.Request.Header.ContentType()) == "" {
		return defaultContentType, nil
	}
	baseCT, _, err := mime.ParseMediaType(string(fctx.Request.Header.ContentType()))
	if err != nil {
		return "", errs.Wrapf(err, codec.SerializationTypeUnsupported, "invalid content type:%s",
			fctx.Request.Header.ContentType())
	}
	if baseCT == "multipart/form-data" {
		return defaultContentType, nil
	}
	return string(fctx.Request.Header.ContentType()), nil
}

// getSerializationType gets the serialization type
func getSerializationType(ctx context.Context) (int, error) {
	fCtx := chttp.RequestContext(ctx)
	if fCtx == nil {
		return 0, errs.New(gerrs.ErrWrongContext, "not an HTTP request")
	}
	// Get serialization type
	contentType := string(fCtx.Request.Header.ContentType())
	if contentType == "" {
		contentType = "application/json"
	}
	st, err := ctrpc.GetSerializationType(contentType)
	if err != nil {
		return 0, gerrs.Wrap(err, "error getting serialization type")
	}
	return st, nil
}
