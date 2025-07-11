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

package fhttp

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prashantv/gostub"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/core/entity"
	"trpc.group/trpc-go/trpc-gateway/core/service/protocol"
	pmock "trpc.group/trpc-go/trpc-gateway/core/service/protocol/mock"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/client/mockclient"
	"trpc.group/trpc-go/trpc-go/codec"
)

type fakeRouter struct{}

func (r *fakeRouter) GetMatchRouter(context.Context) (*entity.TargetService, error) {
	return &entity.TargetService{}, nil
}

// LoadRouter loads the configuration
func (r *fakeRouter) LoadRouterConf(string) error {
	return nil
}

// InitRouterConfig initializes the configuration
func (r *fakeRouter) InitRouterConfig(context.Context, *entity.ProxyConfig) (err error) {
	return nil
}

func TestHandler_HTTPHandler(t *testing.T) {
	h.SetRouter(&fakeRouter{})
	ctx := trpc.BackgroundContext()
	ctx = http.WithRequestContext(ctx, &fasthttp.RequestCtx{})
	ctx, _ = codec.WithNewMessage(ctx)
	_ = h.HTTPHandler(ctx)
	assert.NotNil(t, ctx)
}

type optMatcher struct{}

var optCount int
var gotMyHeader bool

func (g *optMatcher) Matches(x interface{}) bool {
	optCount++
	o := &client.Options{}
	opt, ok := x.(client.Option)
	if !ok {
		return true
	}
	opt(o)
	if val := o.MetaData["my-header"]; string(val) == "val" {
		gotMyHeader = true
	}
	if optCount >= 11 && !gotMyHeader {
		return false
	}
	return true
}

func (g *optMatcher) String() string {
	return ""
}

func Test_handler_HTTPHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	stub := gostub.New()
	defer stub.Reset()
	mockClient := mockclient.NewMockClient(ctrl)
	stub.Stub(&client.DefaultClient, mockClient)

	mockCliProtocol := pmock.NewMockCliProtocolHandler(ctrl)
	protocol.RegisterCliProtocolHandler("fasthttp", mockCliProtocol)
	mockCliProtocol.EXPECT().GetCliOptions(gomock.Any()).Return(nil, nil).Times(2)
	mockCliProtocol.EXPECT().WithCtx(gomock.Any()).Return(context.Background(), nil).Times(2)
	mockCliProtocol.EXPECT().TransReqBody(gomock.Any()).Return(gomock.Any(), nil).Times(2)
	mockCliProtocol.EXPECT().TransRspBody(gomock.Any()).Return(gomock.Any(), nil).Times(2)
	mockCliProtocol.EXPECT().HandleErr(gomock.Any(), gomock.Any()).Return(errors.New("err")).Times(1)
	mockCliProtocol.EXPECT().HandleRspBody(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	ctx := context.Background()
	ctx, gMsg := gwmsg.WithNewGWMessage(ctx)
	defer gwmsg.PutBackGwMessage(gMsg)

	// Forwarding success
	var mList []interface{}
	for i := 0; i < 10; i++ {
		mList = append(mList, &optMatcher{})
	}

	mockClient.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any(), mList...).Return(nil)
	fCtx := &fasthttp.RequestCtx{}
	fCtx.Request.URI().SetPath("/user/info")
	fCtx.Request.Header.Set("my-header", "val")

	ctx = http.WithRequestContext(ctx, fCtx)
	ctx, msg := codec.WithNewMessage(ctx)
	md := codec.MetaData{
		"a": []byte("b"),
	}
	msg.WithCalleeMethod("/user/")
	msg.WithServerMetaData(md)
	cliConfig := &client.BackendConfig{
		DisableServiceRouter: true,
		Target:               "target",
		Protocol:             "fasthttp",
	}
	gMsg.WithTargetService(cliConfig)
	err := h.HTTPHandler(ctx)
	assert.Nil(t, err)

	// fCtx is empty
	err = h.HTTPHandler(context.Background())
	assert.NotNil(t, err)
	// Forwarding failure
	mockClient.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("err"))
	err = h.HTTPHandler(ctx)
	assert.NotNil(t, err)

	cliConfig.Protocol = "invalid"
	err = h.HTTPHandler(ctx)
	assert.NotNil(t, err)
	cliConfig.Protocol = "fasthttp"

	mockCliProtocol.EXPECT().WithCtx(gomock.Any()).Return(context.Background(), nil).Times(1)
	mockCliProtocol.EXPECT().GetCliOptions(gomock.Any()).Return(nil, errors.New("err"))
	err = h.HTTPHandler(ctx)
	assert.NotNil(t, err)

	mockCliProtocol.EXPECT().WithCtx(gomock.Any()).Return(context.Background(), nil).Times(1)
	mockCliProtocol.EXPECT().GetCliOptions(gomock.Any()).Return(nil, nil)
	mockCliProtocol.EXPECT().TransReqBody(gomock.Any()).Return(gomock.Any(), errors.New("err"))
	err = h.HTTPHandler(ctx)
	assert.NotNil(t, err)

	mockCliProtocol.EXPECT().WithCtx(gomock.Any()).Return(context.Background(), nil).Times(1)
	mockCliProtocol.EXPECT().GetCliOptions(gomock.Any()).Return(nil, nil)
	mockCliProtocol.EXPECT().TransReqBody(gomock.Any()).Return(gomock.Any(), nil)
	mockCliProtocol.EXPECT().TransRspBody(gomock.Any()).Return(gomock.Any(), errors.New("err"))
	err = h.HTTPHandler(ctx)
	assert.NotNil(t, err)
}
