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

package gwmsg_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	mockgwmsg "trpc.group/trpc-go/trpc-gateway/common/gwmsg/mock"
	"trpc.group/trpc-go/trpc-go/client"
)

func TestGwMsg(t *testing.T) {
	// Create a new gwMsg
	ctx, msg := gwmsg.WithNewGWMessage(context.Background())
	mockFunc(ctx)
	// Get the target server
	assert.Equal(t, "target", msg.TargetService().Target)

	// Get router id
	assert.Equal(t, "routerID", msg.RouterID())

	// Get upstream latency
	assert.EqualValues(t, 1, msg.UpstreamLatency())
	// Get plugin configuration
	iConfig := msg.PluginConfig("demo")
	assert.NotNil(t, iConfig)
	dc, ok := iConfig.(*demoConfig)
	assert.True(t, ok)
	assert.Equal(t, dc.Name, "props")

	// Get UpstreamAddr
	assert.Equal(t, "0.0.0.0", msg.UpstreamAddr())
	assert.Equal(t, "/user/info", msg.UpstreamMethod())
	assert.Equal(t, "rsp header", msg.UpstreamRspHead().(string))
	assert.Equal(t, 2, len(msg.TRPCClientOpts()))

	gwmsg.PutBackGwMessage(msg)
	assert.Nil(t, msg.TargetService())
	iConfig = msg.PluginConfig("demo")
	assert.Nil(t, iConfig)

	gwmsg.PutBackGwMessage(&mockgwmsg.MockGwMsg{})
}

func mockFunc(ctx context.Context) {
	// Set the target service
	gwmsg.GwMessage(ctx).WithTargetService(&client.BackendConfig{Target: "target"})
	// Set plugin configuration
	gwmsg.GwMessage(ctx).WithPluginConfig("demo", &demoConfig{
		Name: "props",
	})
	// Set router id
	gwmsg.GwMessage(ctx).WithRouterID("routerID")
	// Set upstream latency
	gwmsg.GwMessage(ctx).WithUpstreamLatency(1)

	gwmsg.GwMessage(ctx).WithUpstreamAddr("0.0.0.0")

	gwmsg.GwMessage(ctx).WithUpstreamMethod("/user/info")

	gwmsg.GwMessage(ctx).WithUpstreamRspHead("rsp header")
	gwmsg.GwMessage(ctx).WithTRPCClientOpts([]client.Option{
		func(options *client.Options) {
		},
	})
	gwmsg.GwMessage(ctx).WithTRPCClientOpts([]client.Option{
		func(options *client.Options) {
		},
	})
}

type demoConfig struct {
	Name string
}

func TestGwMessage(t *testing.T) {
	ctx := context.Background()
	gwmsg.GwMessage(ctx)
}
