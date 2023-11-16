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

// Package gwmsg defines gateway message
package gwmsg

import (
	"trpc.group/trpc-go/trpc-go/client"
)

//go:generate mockgen -destination=./mock/mock_msg.go  . GwMsg

// GwMsg Gateway Msg interface, used for parameter sharing
type GwMsg interface {
	// WithTargetService sets target service
	WithTargetService(cli *client.BackendConfig)
	// TargetService returns target service
	TargetService() *client.BackendConfig

	// WithPluginConfig sets plugin configuration
	WithPluginConfig(name string, config interface{})
	// PluginConfig returns plugin configuration, do not make concurrent calls
	PluginConfig(name string) interface{}

	// WithRouterID sets router ID
	WithRouterID(routerID string)
	// RouterID returns router ID
	RouterID() string

	// WithUpstreamLatency sets upstream latency
	WithUpstreamLatency(latency int64)
	// UpstreamLatency returns upstream latency
	UpstreamLatency() int64

	// WithUpstreamAddr sets upstream address
	WithUpstreamAddr(add string)
	// UpstreamAddr returns upstream address
	UpstreamAddr() string

	// WithUpstreamMethod sets upstream method
	WithUpstreamMethod(method string)
	// UpstreamMethod returns upstream method
	UpstreamMethod() string

	// WithUpstreamRspHead sets upstream ClientRspHead
	WithUpstreamRspHead(rspHead interface{})
	// UpstreamRspHead returns upstream ClientRspHead
	UpstreamRspHead() interface{}

	// WithTRPCClientOpts sets trpc client options
	WithTRPCClientOpts(opts []client.Option)
	// TRPCClientOpts returns trpc client options
	TRPCClientOpts() []client.Option
}
