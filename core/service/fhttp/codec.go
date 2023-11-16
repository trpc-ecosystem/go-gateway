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

package fhttp

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	ghttp "trpc.group/trpc-go/trpc-gateway/common/http"
	gtrpc "trpc.group/trpc-go/trpc-gateway/common/trpc"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/log"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"
)

var (
	// DefaultServerCodec is the default server codec
	DefaultServerCodec = &ServerCodec{
		ErrHandler: defaultErrHandler,
		RspHandler: defaultRspHandler,
	}
	// DefaultClientCodec is the default client codec
	DefaultClientCodec = &ClientCodec{}
)

func init() {
	codec.Register("fasthttp", DefaultServerCodec, DefaultClientCodec)
}

// ErrorHandler is the error handler function for fasthttp server response. By default, it puts the error code in the
// header, but it can be replaced with a specific implementation.
type ErrorHandler func(ctx context.Context, e *errs.Error)

// defaultErrHandler is the default error handler
func defaultErrHandler(ctx context.Context, e *errs.Error) {
	if e == nil {
		return
	}
	fctx := ghttp.RequestContext(ctx)
	if fctx == nil {
		return
	}

	if e.Type == errs.ErrorTypeFramework {
		fctx.Response.Header.Set(http.TrpcFrameworkErrorCode, strconv.Itoa(int(e.Code)))
	} else {
		fctx.Response.Header.Set(http.TrpcUserFuncErrorCode, strconv.Itoa(int(e.Code)))
	}

	// In non-production environment, return error message for debugging purposes
	if !gtrpc.DefaultIsProduction() {
		// Avoid CRLF injection vulnerability
		errMsg := strings.Replace(e.Msg, "\r", "\\r", -1)
		errMsg = strings.Replace(errMsg, "\n", "\\n", -1)
		fctx.Response.Header.Set(http.TrpcErrorMessage, errMsg)
	}

	// Adjust HTTP status code in case of error
	if fctx.Response.StatusCode() != fasthttp.StatusOK {
		return
	}

	fctx.SetStatusCode(gerrs.GetHTTPStatus(e.Code))
}

// ResponseHandler is the response handler function for fasthttp server response. By default, it directly returns the
// response body, but it can be replaced with a specific implementation.
type ResponseHandler func(ctx context.Context, rspbody []byte) error

// defaultRspHandler is the default response handler
var defaultRspHandler = func(ctx context.Context, rspbody []byte) error {
	if len(rspbody) == 0 {
		return nil
	}
	if _, err := ghttp.RequestContext(ctx).Write(rspbody); err != nil {
		return gerrs.Wrap(err, "http write response err")
	}
	return nil
}

// setGatewayHeader sets the gateway response headers
func setGatewayHeader(ctx context.Context) error {
	fctx := ghttp.RequestContext(ctx)
	if fctx == nil {
		return errs.New(gerrs.ErrWrongContext, "invalid http context")
	}
	// For security reasons, do not pass internal transmission information
	fctx.Response.Header.Del(http.TrpcTransInfo)
	// Upstream latency
	gwMsg := gwmsg.GwMessage(ctx)
	upstreamLatency := gwMsg.UpstreamLatency()
	fctx.Response.Header.Set(ghttp.XUpstreamLatencyHeader, fmt.Sprint(upstreamLatency))
	// Proxy latency
	fctx.Response.Header.Set(ghttp.XProxyLatencyHeader,
		fmt.Sprint(time.Since(fctx.ConnTime()).Milliseconds()-upstreamLatency))
	// Set router_id
	if !gtrpc.DefaultIsProduction() {
		fctx.Response.Header.Set(ghttp.XRouterIDHeader, gwMsg.RouterID())
	}
	return nil
}

// ServerCodec is the HTTP server codec
type ServerCodec struct {
	// ErrHandler is the error code handler function. By default, it fills the error code in the header.
	// The business can replace it with a specific implementation using
	// fhttp.DefaultServerCodec.ErrHandler = func(rsp, req, err) {}.
	ErrHandler ErrorHandler

	// RspHandler is the response data handler function. By default, it directly returns the data.
	// The business can customize this method to shape the response data using
	// fhttp.DefaultServerCodec.RspHandler = func(rsp, req, rspbody) {}.
	RspHandler ResponseHandler
}

// Decode decodes the incoming request
func (sc *ServerCodec) Decode(msg codec.Msg, _ []byte) (ret []byte, err error) {
	ctx := ghttp.RequestContext(msg.Context())
	if ctx == nil {
		return nil, errs.New(gerrs.ErrWrongContext, "server decode missing fasthttp request in context")
	}

	msg.WithCalleeMethod(string(ctx.Path()))

	if err := sc.setReqHeader(ctx, msg); err != nil {
		return nil, gerrs.Wrap(err, "set_req_header_err")
	}

	// Upstream
	if msg.CallerServiceName() == "" {
		msg.WithCallerServiceName("trpc.http.upserver.upservice")
	}

	// Self
	if msg.CalleeServiceName() == "" {
		msg.WithCalleeServiceName(fmt.Sprintf("trpc.http.%s.service", path.Base(os.Args[0])))
	}

	return
}

// Encode encodes theresponse for sending back to the client
func (sc *ServerCodec) Encode(msg codec.Msg, rspbody []byte) (ret []byte, err error) {
	if err := setGatewayHeader(msg.Context()); err != nil {
		log.ErrorContextf(msg.Context(), "set gateway header err:%s", err)
	}

	// 1. Handle exceptional cases first. If the server returns an error, stop processing the response data.
	if e := msg.ServerRspErr(); e != nil {
		if sc.ErrHandler != nil {
			sc.ErrHandler(msg.Context(), e)
		}
		return
	}

	// 2. Handle normal response data
	if sc.RspHandler != nil {
		if err := sc.RspHandler(msg.Context(), rspbody); err != nil {
			return nil, gerrs.Wrap(err, "rsp_handler_err")
		}
	}

	return
}

// setReqHeader sets the request headers
func (sc *ServerCodec) setReqHeader(ctx *fasthttp.RequestCtx, msg codec.Msg) error {
	trpcReq := &trpcpb.RequestProtocol{}
	msg.WithServerReqHead(trpcReq)
	msg.WithServerRspHead(trpcReq)

	trpcReq.Func = []byte(msg.ServerRPCName())
	trpcReq.ContentType = uint32(msg.SerializationType())
	trpcReq.ContentEncoding = uint32(msg.CompressType())

	if v := ctx.Request.Header.Peek(http.TrpcVersion); v != nil {
		i, _ := strconv.Atoi(string(v))
		trpcReq.Version = uint32(i)
	}
	if v := ctx.Request.Header.Peek(http.TrpcCallType); v != nil {
		i, _ := strconv.Atoi(string(v))
		trpcReq.CallType = uint32(i)
	}
	if v := ctx.Request.Header.Peek(http.TrpcMessageType); v != nil {
		i, _ := strconv.Atoi(string(v))
		trpcReq.MessageType = uint32(i)
	}
	if v := ctx.Request.Header.Peek(http.TrpcRequestID); v != nil {
		i, _ := strconv.Atoi(string(v))
		trpcReq.RequestId = uint32(i)
	}
	if v := ctx.Request.Header.Peek(http.TrpcTimeout); v != nil {
		i, _ := strconv.Atoi(string(v))
		trpcReq.Timeout = uint32(i)
		msg.WithRequestTimeout(time.Millisecond * time.Duration(i))
	}
	if v := ctx.Request.Header.Peek(http.TrpcCaller); v != nil {
		trpcReq.Caller = v
		msg.WithCallerServiceName(string(v))
	}
	if v := ctx.Request.Header.Peek(http.TrpcCallee); v != nil {
		trpcReq.Callee = v
		msg.WithCalleeServiceName(string(v))
	}

	msg.WithDyeing((trpcReq.GetMessageType() & uint32(trpcpb.TrpcMessageType_TRPC_DYEING_MESSAGE)) != 0)

	setForwardedFor(ctx)

	if v := ctx.Request.Header.Peek(http.TrpcTransInfo); v != nil {
		return setTransInfo(trpcReq, msg, v)
	}
	return nil
}

// setForwardedFor sets the x-forwarded-for header
func setForwardedFor(ctx *fasthttp.RequestCtx) {
	clientIP, _, err := net.SplitHostPort(ctx.RemoteAddr().String())
	if err != nil {
		// Ignore error
		return
	}
	// If we aren't the first proxy, retain prior X-Forwarded-For information as a comma+space separated list and fold
	// multiple headers into one.
	prior := ctx.Request.Header.Peek(fasthttp.HeaderXForwardedFor)
	if len(prior) > 0 {
		clientIP = string(prior) + ", " + clientIP
	}
	ctx.Request.Header.Set(fasthttp.HeaderXForwardedFor, clientIP)
}

// setTransInfo sets the transInfo
func setTransInfo(trpcReq *trpcpb.RequestProtocol, msg codec.Msg, val []byte) error {
	m := make(map[string]string)
	if err := codec.Unmarshal(codec.SerializationTypeJSON, val, &m); err != nil {
		return gerrs.Wrap(err, "unmarshal_trans_info_err")
	}
	trpcReq.TransInfo = make(map[string][]byte)
	// Since HTTP headers can only transmit plaintext strings, but trpc transinfo is binary data, it needs to be
	// protected with base64 encoding.
	for k, v := range m {
		decoded, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			decoded = []byte(v)
		}
		trpcReq.TransInfo[k] = decoded

		if k == http.TrpcEnv {
			msg.WithEnvTransfer(string(decoded))
		}
		if k == http.TrpcDyeingKey {
			msg.WithDyeingKey(string(decoded))
		}
	}
	msg.WithServerMetaData(trpcReq.GetTransInfo())
	return nil
}

// ClientCodec encodes and decodes HTTP client requests
type ClientCodec struct{}

// Encode sets the metadata for the HTTP client request. This is the proxy layer and does not perform header settings.
func (c *ClientCodec) Encode(msg codec.Msg, _ []byte) (ret []byte, err error) {
	if msg.CallerServiceName() == "" {
		msg.WithCallerServiceName(fmt.Sprintf("trpc.http.%s.service", path.Base(os.Args[0])))
	}

	return
}

// Decode parses the metadata from the HTTP client response. This is the proxy layer and does not perform any work.
func (c *ClientCodec) Decode(codec.Msg, []byte) (rspBody []byte, err error) {
	return
}
