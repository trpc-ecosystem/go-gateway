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

package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"trpc.group/trpc-go/trpc-gateway/common/convert"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/transport"
)

// Protocol is the protocol name
const Protocol = "grpc"

func init() {
	transport.RegisterClientTransport(Protocol, DefaultClientTransport)
}

// DefaultClientTransport is the default client transport layer
var DefaultClientTransport = &ClientTransport{
	ConnectionPool: &Pool{},
}

// ClientTransport implements the transport.ClientTransport interface of trpc-go, using native grpc transport layer
// instead of trpc-go transport layer
type ClientTransport struct {
	ConnectionPool ConnPool
	StreamClient   grpc.ClientStream
}

// RoundTrip is the method that implements transport.ClientTransport, invoking native grpc client code
func (c *ClientTransport) RoundTrip(ctx context.Context, _ []byte,
	roundTripOpts ...transport.RoundTripOption) (rsp []byte, err error) {
	// Use grpc client to call remote server method
	header := Head(ctx)
	if header == nil {
		return nil, errs.New(gerrs.ErrGatewayUnknown, "get no grpc header round trip")
	}
	// Default values
	opts := &transport.RoundTripOptions{}

	// Write the incoming func options to the opts field
	for _, o := range roundTripOpts {
		o(opts)
	}

	msg := codec.Message(ctx)
	// Get the timeout setting
	timeout := msg.RequestTimeout()
	// Get metadata from ctx and set client metadata using grpc method TODO Pass metadata
	ctx, err = setGRPCMetadata(ctx, header)
	if err != nil {
		return nil, gerrs.Wrap(err, "set grpc metadata err")
	}

	// Get metadata from the server
	md := &metadata.MD{}
	var callOpts []grpc.CallOption
	callOpts = append(callOpts, grpc.Header(md))

	// Get grpc connection from the connection pool
	conn, err := c.ConnectionPool.Get(opts.Address, timeout)
	if err != nil {
		return nil, errs.WrapFrameError(err, errs.RetClientConnectFail,
			"grpc client transport RoundTrip get conn fail")
	}
	if err = conn.Invoke(ctx, msg.ClientRPCName(),
		header.Req, ctx, callOpts...); err != nil {
		if status.Code(err) == codes.DeadlineExceeded {
			return nil, errs.WrapFrameError(err, errs.RetClientTimeout,
				"grpc client transport RoundTrip timeout")
		}
		if status.Code(err) == codes.Canceled {
			return nil, errs.WrapFrameError(err, errs.RetClientCanceled,
				"grpc client transport RoundTrip canceled")
		}
		return nil, errs.WrapFrameError(err, errs.RetClientNetErr,
			"grpc client transport RoundTrip")
	}
	// Write the server's metadata into ctx for upper layer to access
	header.InMetadata = *md
	log.DebugContextf(ctx, "in metadata:%s", convert.ToJSONStr(md))
	return nil, nil
}

// setGRPCMetadata sets the grpc Header information into metadata
func setGRPCMetadata(ctx context.Context, header *Header) (context.Context, error) {
	// Set grpc md to ctx for the sender to use
	var kv []string
	for k, vals := range header.OutMetadata {
		for _, v := range vals {
			kv = append(kv, k, v)
		}
	}
	if kv != nil {
		ctx = metadata.AppendToOutgoingContext(ctx, kv...)
	}
	return ctx, nil
}
