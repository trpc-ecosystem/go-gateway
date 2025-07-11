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

package gwmsg

import (
	"context"
	"sync"

	"trpc.group/trpc-go/trpc-go/client"
)

// gwMsg is the context of gateway
type gwMsg struct {
	cli             *client.BackendConfig
	pluginConfig    map[string]interface{}
	routerID        string
	upstreamLatency int64
	upstreamAddr    string
	upstreamMethod  string
	upstreamRspHead interface{}
	tRPCClientOpts  []client.Option
}

// WithPluginConfig sets plugin configuration
func (gm *gwMsg) WithPluginConfig(name string, config interface{}) {
	if gm.pluginConfig == nil {
		gm.pluginConfig = make(map[string]interface{})
	}
	gm.pluginConfig[name] = config
}

// PluginConfig returns plugin configuration, do not make concurrent calls
func (gm *gwMsg) PluginConfig(name string) interface{} {
	if gm.pluginConfig == nil {
		return nil
	}
	return gm.pluginConfig[name]
}

// WithTargetService sets target service
func (gm *gwMsg) WithTargetService(cli *client.BackendConfig) {
	gm.cli = cli
}

// TargetService returns target service
func (gm *gwMsg) TargetService() *client.BackendConfig {
	return gm.cli
}

// WithRouterID Set router ID
func (gm *gwMsg) WithRouterID(routerID string) {
	gm.routerID = routerID
}

// RouterID Get router ID
func (gm *gwMsg) RouterID() string {
	return gm.routerID
}

// WithUpstreamLatency sets upstream latency
func (gm *gwMsg) WithUpstreamLatency(latency int64) {
	gm.upstreamLatency = latency
}

// UpstreamLatency returns upstream latency
func (gm *gwMsg) UpstreamLatency() int64 {
	return gm.upstreamLatency
}

// WithUpstreamAddr sets upstream address
func (gm *gwMsg) WithUpstreamAddr(addr string) {
	gm.upstreamAddr = addr
}

// UpstreamAddr returns upstream address
func (gm *gwMsg) UpstreamAddr() string {
	return gm.upstreamAddr
}

// WithUpstreamMethod sets upstream method
func (gm *gwMsg) WithUpstreamMethod(method string) {
	gm.upstreamMethod = method
}

// UpstreamMethod returns upstream method
func (gm *gwMsg) UpstreamMethod() string {
	return gm.upstreamMethod
}

// WithUpstreamRspHead sets upstream client response header
func (gm *gwMsg) WithUpstreamRspHead(rspHead interface{}) {
	gm.upstreamRspHead = rspHead
}

// UpstreamRspHead returns upstream ClientRspHead
func (gm *gwMsg) UpstreamRspHead() interface{} {
	return gm.upstreamRspHead
}

// WithTRPCClientOpts sets trpc client options
func (gm *gwMsg) WithTRPCClientOpts(opts []client.Option) {
	if gm.tRPCClientOpts == nil {
		gm.tRPCClientOpts = opts
		return
	}
	gm.tRPCClientOpts = append(gm.tRPCClientOpts, opts...)
}

// TRPCClientOpts returns trpc client options
func (gm *gwMsg) TRPCClientOpts() []client.Option {
	return gm.tRPCClientOpts
}

// gwMsgPool 网关消息对象池，用来进行对象复用，减少GC
var gwMsgPool = sync.Pool{
	New: func() interface{} {
		return &gwMsg{}
	},
}

// ContextKey 重定义一下key类型，防止字符串定义冲突
type ContextKey string

// ContextKeyGwMessage 网关Msg key
const ContextKeyGwMessage ContextKey = "TRPC_GATEWAY_MESSAGE"

// WithNewGWMessage create a new empty message, and put it into ctx
func WithNewGWMessage(ctx context.Context) (context.Context, GwMsg) {
	m, ok := gwMsgPool.Get().(*gwMsg)
	if !ok {
		return ctx, &gwMsg{}
	}
	ctx = context.WithValue(ctx, ContextKeyGwMessage, m)
	return ctx, m
}

// PutBackGwMessage return struct Message to sync pool,
// and reset all the members of Message to default.
func PutBackGwMessage(sourceMsg GwMsg) {
	m, ok := sourceMsg.(*gwMsg)
	if !ok {
		return
	}
	m.resetDefault()
	gwMsgPool.Put(m)
}

// resetDefault reset all fields of msg to default value.
func (gm *gwMsg) resetDefault() {
	gm.cli = nil
	gm.pluginConfig = nil
	gm.routerID = ""
	gm.upstreamLatency = 0
	gm.upstreamAddr = ""
	gm.upstreamMethod = ""
	gm.upstreamRspHead = nil
	gm.tRPCClientOpts = nil
}

// GwMessage returns the message of context.
func GwMessage(ctx context.Context) GwMsg {
	val := ctx.Value(ContextKeyGwMessage)
	m, ok := val.(*gwMsg)
	if !ok {
		return &gwMsg{}
	}
	return m
}
