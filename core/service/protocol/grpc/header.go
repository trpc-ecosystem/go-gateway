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
	"strings"

	"google.golang.org/grpc/metadata"
	"trpc.group/trpc-go/trpc-go/log"
)

// Header is stored in the context to communicate with trpc
type Header struct {
	Req         []byte      // request
	Rsp         []byte      // response
	InMetadata  metadata.MD // metadata from client
	OutMetadata metadata.MD // metadata sent to client
}

// ContextKey 定义 grpc 的 contextKey
type ContextKey string

// ContextKeyHeader is the key for the GRPC header information in the context
const ContextKeyHeader = ContextKey("TRPC_GATEWAY_GRPC_HEADER")

// Head retrieves the GRPC header information
func Head(ctx context.Context) *Header {
	if header, ok := ctx.Value(ContextKeyHeader).(*Header); ok {
		return header
	}
	return nil
}

// WithHeader sets the GRPC header information in the context
func WithHeader(ctx context.Context, header *Header) context.Context {
	return context.WithValue(ctx, ContextKeyHeader, header)
}

// WithServerGRPCMetadata is used for trpc-go server calls to send metadata
func WithServerGRPCMetadata(ctx context.Context, key string, value []string) {
	// Cannot include connection metadata, otherwise grpc will upgrade to the http2 protocol
	if strings.ToLower(key) == "connection" {
		return
	}
	header := Head(ctx)
	if header == nil {
		log.DebugContextf(ctx, "with grpc metadata nil")
		return
	}
	if header.OutMetadata == nil {
		header.OutMetadata = metadata.MD{}
	}
	header.OutMetadata.Set(key, value...)
}
