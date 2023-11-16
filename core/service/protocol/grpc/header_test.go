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
	"testing"
)

func TestWithServerGRPCMetadata(t *testing.T) {
	WithServerGRPCMetadata(context.Background(), "connection", []string{"a"})
	WithServerGRPCMetadata(context.Background(), "name", []string{"a"})
	ctx := WithHeader(context.Background(), &Header{
		Req:         nil,
		Rsp:         nil,
		InMetadata:  nil,
		OutMetadata: nil,
	})
	WithServerGRPCMetadata(ctx, "name", []string{"a"})
}
