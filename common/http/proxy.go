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

package http

const (
	// XUpstreamLatencyHeader is the upstream latency response header.
	XUpstreamLatencyHeader = "X-Upstream-Latency"
	// XProxyLatencyHeader is the gateway latency response header.
	XProxyLatencyHeader = "X-Proxy-Latency"
	// XRouterIDHeader is the router ID.
	XRouterIDHeader = "X-Router-Id"
	// GatewayName is the name of the gateway service.
	GatewayName = "tRPC-Gateway"
)
