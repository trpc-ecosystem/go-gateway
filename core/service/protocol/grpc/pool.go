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
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"trpc.group/trpc-go/trpc-go/errs"
)

// ConnPool connection pool
//
//go:generate mockgen -destination=./mock/connpool.go -package=mock_grpc . ConnPool
type ConnPool interface {
	Get(address string, timeout time.Duration) (grpc.ClientConnInterface, error)
}

// Pool implements a simple grpc connection pool
type Pool struct {
	connections sync.Map
}

// Get retrieves an available grpc client connection from the connection pool
func (p *Pool) Get(address string, timeout time.Duration) (grpc.ClientConnInterface, error) {
	// TODO Consider timeout when indexing the connection pool
	if v, ok := p.connections.Load(address); ok {
		return v.(grpc.ClientConnInterface), nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()), // TODO 从ctx中获取证书相关配置，支持tls通讯
		grpc.WithDefaultCallOptions(grpc.CallContentSubtype("json")),
	)
	if err != nil {
		return nil, errs.NewFrameError(errs.RetClientConnectFail, err.Error())
	}
	v, loaded := p.connections.LoadOrStore(address, conn)
	if !loaded {
		return conn, nil
	}
	return v.(grpc.ClientConnInterface), nil
}
