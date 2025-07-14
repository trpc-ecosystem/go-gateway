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

// Package errs API gateway framework error related definitions, related methods
package errs

import trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

const (
	// Success status code
	Success = 0
	// ErrGatewayUnknown Distinguishing between unknown errors in the gateway and unknown error codes in trpc
	ErrGatewayUnknown = trpcpb.TrpcRetCode(999)

	// ErrWrongConfig Configuration error
	ErrWrongConfig = trpcpb.TrpcRetCode(1001)

	// ErrPathNotFound No route matched
	ErrPathNotFound = trpcpb.TrpcRetCode(1002)

	// ErrWrongContext Context type error (non-fasthttp context)
	ErrWrongContext = trpcpb.TrpcRetCode(1003)

	// ErrTargetServiceNotFound Failed to obtain the target server
	ErrTargetServiceNotFound = trpcpb.TrpcRetCode(1004)

	// ErrContextNoServiceVal Failed to obtain service information from the context
	ErrContextNoServiceVal = trpcpb.TrpcRetCode(1005)

	// ErrPluginConfigNotFound Failed to obtain plugin configuration from the context
	ErrPluginConfigNotFound = trpcpb.TrpcRetCode(1006)

	// ErrInvalidPluginConfig Plugin configuration error
	ErrInvalidPluginConfig = trpcpb.TrpcRetCode(1007)

	// ErrUpstreamRspErr upstream HTTP status code is not 200
	ErrUpstreamRspErr = trpcpb.TrpcRetCode(1008)

	// ErrInvalidReq Illegal request, protocol error, etc.
	ErrInvalidReq = trpcpb.TrpcRetCode(1009)

	// ErrUnSupportProtocol Unsupported protocol type
	ErrUnSupportProtocol = trpcpb.TrpcRetCode(1010)

	// ErrProtocolTrans Protocol conversion failed
	ErrProtocolTrans = trpcpb.TrpcRetCode(1011)

	// ErrConnClosed Connection pool closed
	ErrConnClosed = trpcpb.TrpcRetCode(1012)
)

const (
	// GatewayERRKey Reporting key for abnormal configuration loading to be used for monitoring and alerting
	GatewayERRKey = "trpc_gateway_report"
)
