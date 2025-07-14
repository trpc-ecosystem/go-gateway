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
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/core/entity"
	"trpc.group/trpc-go/trpc-gateway/core/router"
	mockrouter "trpc.group/trpc-go/trpc-gateway/core/router/mock"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/server/mockserver"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"
)

func TestRegisterFastHTTPService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockServer := mockserver.NewMockService(ctrl)
	mockServer.EXPECT().Register(gomock.Any(), gomock.Any()).Return(nil)
	router.RegisterRouter(ProtocolName, router.DefaultFastHTTPRouter)
	RegisterFastHTTPService(mockServer)
}

func Test_generateMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var handlerFunc = func(ctx context.Context) error {
		return nil
	}
	method := generateMethod("*", handlerFunc)
	fCtx := &fasthttp.RequestCtx{}
	ctx := http.WithRequestContext(context.Background(), fCtx)
	mockRouter := mockrouter.NewMockRouter(ctrl)
	h.SetRouter(mockRouter)
	// *client.BackendConfig, []*config.Plugin, error
	mockRouter.EXPECT().GetMatchRouter(gomock.Any()).Return(&entity.TargetService{
		Service:       "",
		BackendConfig: &client.BackendConfig{},
		Weight:        0,
		ReWrite:       "",
		StripPath:     false,
		Plugins: []*entity.Plugin{
			{
				Name: "demo",
				Type: "gateway",
				Props: map[string]interface{}{
					"data": "xxx",
				},
			},
			{
				Name: "invalid_filter", // Invalid filter
				Type: "gateway",
				Props: map[string]interface{}{
					"data": "xxx",
				},
			},
		},
		Filters: nil,
	}, nil)
	var filterFunc = func(ctx context.Context, req interface{}, handler filter.ServerHandleFunc) (interface{}, error) {
		return handler(ctx, req)
	}
	filter.Register("demo", filterFunc, nil)
	tConfig := &trpc.Config{}
	tConfig.Server.Filter = []string{"auth"}
	trpc.SetGlobalConfig(tConfig)
	filter.Register("auth", filterFunc, nil)
	// Execution success
	_, err := method.Func(nil, ctx, nil)
	assert.Nil(t, err)
	// Pre-processing validation failure
	DefaultPreProcessRoute = func(ctx context.Context) error {
		return errs.New(2333, "pre process router err")
	}

	// Execution failure
	_, err = method.Func(nil, ctx, nil)
	assert.Equal(t, trpcpb.TrpcRetCode(2333), errs.Code(err))

	DefaultPreProcessRoute = func(ctx context.Context) error {
		return nil
	}
	// Routing failure
	mockRouter.EXPECT().GetMatchRouter(gomock.Any()).Return(nil, errors.New("err"))
	// Execution failure
	_, err = method.Func(nil, ctx, nil)
	assert.NotNil(t, err)

	// Function execution error
	mockRouter.EXPECT().GetMatchRouter(gomock.Any()).Return(
		&entity.TargetService{
			Service:       "",
			BackendConfig: &client.BackendConfig{},
			Weight:        0,
			ReWrite:       "",
			StripPath:     false,
			Plugins: []*entity.Plugin{
				{
					Name: "demo",
					Type: "gateway",
					Props: map[string]interface{}{
						"data": "xxx",
					},
				},
				{
					Name: "invalid_filter", // Invalid filter
					Type: "gateway",
					Props: map[string]interface{}{
						"data": "xxx",
					},
				},
			},
			Filters: []filter.ServerFilter{},
		}, nil)
	var handlerFunc2 = func(ctx context.Context) error {
		return errors.New("handler err")
	}
	method = generateMethod("", handlerFunc2)
	_, err = method.Func(nil, ctx, nil)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "handler err")

	// Plugin execution failure
	mockRouter.EXPECT().GetMatchRouter(gomock.Any()).Return(
		&entity.TargetService{
			Service:       "",
			BackendConfig: &client.BackendConfig{},
			Weight:        0,
			ReWrite:       "",
			StripPath:     false,
			Plugins: []*entity.Plugin{
				{
					Name: "demo",
					Type: "gateway",
					Props: map[string]interface{}{
						"data": "xxx",
					},
				},
				{
					Name: "invalid_filter", // Invalid filter
					Type: "gateway",
					Props: map[string]interface{}{
						"data": "xxx",
					},
				},
			},
			Filters: []filter.ServerFilter{
				func(ctx context.Context, req interface{}, handler filter.ServerHandleFunc) (interface{}, error) {
					return nil, errors.New("filter err")
				},
			},
		}, nil)

	method = generateMethod("*", handlerFunc)
	_, err = method.Func(nil, ctx, nil)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "filter err")
}

func TestDefaultReportErr(t *testing.T) {
	DefaultReportErr(context.Background(), nil)
	DefaultReportErr(context.Background(), errors.New("err"))
	fctx := &fasthttp.RequestCtx{}
	fctx.Request.Header.Set("Tencent-Leakscan", "xxx")
	ctx := http.WithRequestContext(context.Background(), fctx)

	DefaultReportErr(ctx, errors.New("err"))
}
