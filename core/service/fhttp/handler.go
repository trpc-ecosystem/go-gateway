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
	"time"

	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/core/router"
	"trpc.group/trpc-go/trpc-gateway/core/service/protocol"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	terrs "trpc.group/trpc-go/trpc-go/errs"
)

var h = &handler{}

// handler is a router handler
type handler struct {
	router router.Router
}

// SetRouter sets the router selector.
func (h *handler) SetRouter(r router.Router) {
	h.router = r
}

// HTTPHandler is the default handler function for HTTP requests.
func (h *handler) HTTPHandler(ctx context.Context) error {
	fCtx := http.RequestContext(ctx)
	if fCtx == nil {
		return terrs.New(gerrs.ErrWrongContext, "invalid fasthttp ctx")
	}
	// Clone the message for client monitoring
	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	gwMsg := gwmsg.GwMessage(ctx)
	cliConf := gwMsg.TargetService()
	if cliConf == nil {
		return terrs.New(gerrs.ErrContextNoServiceVal, "get no target service")
	}

	opts := []client.Option{
		client.WithNetwork(cliConf.Network),
		client.WithTarget(cliConf.Target),
		client.WithTimeout(time.Duration(cliConf.Timeout) * time.Millisecond),
		client.WithServiceName(cliConf.ServiceName),
		client.WithProtocol(cliConf.Protocol),
		client.WithCalleeSetName(cliConf.SetName),
		client.WithCalleeEnvName(cliConf.EnvName),
		client.WithNamespace(cliConf.Namespace),
	}
	// Disable service router if specified
	if cliConf.DisableServiceRouter {
		opts = append(opts, client.WithDisableServiceRouter())
	}
	// Set custom options
	opts = append(opts, gwMsg.TRPCClientOpts()...)
	for k, v := range msg.ServerMetaData() {
		opts = append(opts, client.WithMetaData(k, v))
	}
	msg.WithClientRPCName(string(fCtx.Path()))
	msg.WithCalleeServiceName(cliConf.ServiceName)
	msg.WithCalleeMethod(gwmsg.GwMessage(ctx).UpstreamMethod())

	// Protocol conversion
	pt, err := protocol.GetCliProtocolHandler(cliConf.Protocol)
	if err != nil {
		return gerrs.Wrap(err, "get protocol transformer err")
	}
	// set the context header
	ctx, err = pt.WithCtx(ctx)
	if err != nil {
		return gerrs.Wrap(err, "with ctx err")
	}
	// get specific client options for the request
	cliOpts, err := pt.GetCliOptions(ctx)
	if err != nil {
		return gerrs.Wrap(err, "get cli option err")
	}
	opts = append(opts, cliOpts...)

	reqBody, err := pt.TransReqBody(ctx)
	if err != nil {
		return gerrs.Wrap(err, "transform req body err")
	}
	rspBody, err := pt.TransRspBody(ctx)
	if err != nil {
		return gerrs.Wrap(err, "transform rsp body err")
	}
	if err := client.DefaultClient.Invoke(ctx, reqBody, rspBody, opts...); err != nil {
		err = pt.HandleErr(ctx, err)
		if err == nil {
			return nil
		}
		return gerrs.Wrap(err, "invoke proxy err")
	}
	return pt.HandleRspBody(ctx, rspBody)
}
