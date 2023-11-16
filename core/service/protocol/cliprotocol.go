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

package protocol

import (
	"context"
	"sync"

	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/errs"
)

var (
	protocolHandler = make(map[string]CliProtocolHandler)
	lock            sync.RWMutex
)

// RegisterCliProtocolHandler registers downstream protocol handler
func RegisterCliProtocolHandler(protocol string, handler CliProtocolHandler) {
	lock.Lock()
	protocolHandler[protocol] = handler
	lock.Unlock()
}

// GetCliProtocolHandler gets the downstream protocol handler based on the protocol
func GetCliProtocolHandler(protocol string) (CliProtocolHandler, error) {
	lock.RLock()
	defer lock.RUnlock()
	c, ok := protocolHandler[protocol]
	if !ok {
		return nil, errs.Newf(gerrs.ErrUnSupportProtocol, "protocol %s not registered", protocol)
	}
	return c, nil
}

// CliProtocolHandler handles downstream requests based on the downstream client request protocol type
//
//go:generate mockgen -destination=./mock/protocol_mock.go -package=mock . CliProtocolHandler
type CliProtocolHandler interface {
	// WithCtx sets the context header
	WithCtx(ctx context.Context) (context.Context, error)
	// GetCliOptions gets specific client options for the request
	GetCliOptions(ctx context.Context) ([]client.Option, error)
	// TransReqBody transforms the request body
	TransReqBody(ctx context.Context) (interface{}, error)
	// TransRspBody transforms the response body
	TransRspBody(ctx context.Context) (interface{}, error)
	// HandleErr handles error information
	HandleErr(ctx context.Context, err error) error
	// HandleRspBody handles the response
	HandleRspBody(ctx context.Context, rspBody interface{}) error
}
