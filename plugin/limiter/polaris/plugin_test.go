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

package polaris_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/polarismesh/polaris-go/pkg/model"
	"github.com/prashantv/gostub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	gwplugin "trpc.group/trpc-go/trpc-gateway/common/plugin"
	"trpc.group/trpc-go/trpc-gateway/plugin/limiter/polaris"
	mockapi "trpc.group/trpc-go/trpc-gateway/plugin/limiter/polaris/mock"
	trpc "trpc.group/trpc-go/trpc-go"
)

//go:generate mockgen -destination ./mock/polaris_mock.go  github.com/polarismesh/polaris-go/api LimitAPI
//go:generate mockgen -destination ./mock/quota_mock.go  github.com/polarismesh/polaris-go/api QuotaFuture

func TestPluginFactory_Type(t *testing.T) {
	require.Equal(t, gwplugin.DefaultType, (&polaris.PluginFactory{}).Type())
}

func TestPluginFactory_Setup(t *testing.T) {
	pf := polaris.PluginFactory{}
	err := pf.Setup("test", &decoderAlwaysFail{})
	assert.NotNil(t, err)
}

type decoderAlwaysFail struct{}

func (d *decoderAlwaysFail) Decode(interface{}) error {
	return errors.New("decode always fail")
}

type decoder struct {
	timeout    int
	maxRetries int
}

func (d *decoder) Decode(cfg interface{}) error {
	pf, ok := cfg.(*polaris.PluginFactory)
	if !ok {
		return fmt.Errorf("unexpect cfg type %T", reflect.TypeOf(cfg))
	}

	pf.Timeout = d.timeout
	pf.MaxRetries = d.maxRetries

	return nil
}

func TestPluginFactory_CheckConfig(t *testing.T) {
	factory := &polaris.PluginFactory{
		Timeout:    500,
		MaxRetries: 3,
	}
	_ = factory.Setup("", &decoder{
		timeout:    1000,
		maxRetries: 2,
	})

	trpc.GlobalConfig().Server.Service = []*trpc.ServiceConfig{
		{
			Name: "trpc.service",
		},
	}
	err := factory.CheckConfig("", &pluginDecoder{
		Options: &polaris.Options{
			Timeout:        0,
			MaxRetries:     0,
			Service:        "",
			Labels:         nil,
			Namespace:      "",
			LimitedRspBody: "",
		},
	})
	assert.Nil(t, err)
}

type pluginDecoder struct {
	Options *polaris.Options
}

func (d *pluginDecoder) Decode(cfg interface{}) error {
	cfg = d.Options
	return nil
}

func TestInterceptServer(t *testing.T) {
	limiter, err := polaris.New()
	assert.Nil(t, err)
	_, err = limiter.InterceptServer(context.Background(), nil,
		func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
			return nil, nil
		})
	assert.Nil(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	stub := gostub.New()
	defer stub.Reset()
	mockPolaris := mockapi.NewMockLimitAPI(ctrl)
	limiter = &polaris.Limiter{
		Timeout:    500,
		MaxRetries: 3,
		API:        mockPolaris,
	}
	stub.StubFunc(&polaris.New, limiter, nil)
	mockQuota := mockapi.NewMockQuotaFuture(ctrl)
	mockPolaris.EXPECT().GetQuota(gomock.Any()).Return(mockQuota, nil).AnyTimes()
	mockQuota.EXPECT().Get().Return(&model.QuotaResponse{
		Code: 0,
		Info: "",
	}).Times(2)
	mockQuota.EXPECT().Release().Return().AnyTimes()

	factory := &polaris.PluginFactory{
		Timeout:    500,
		MaxRetries: 3,
	}
	_ = factory.Setup("", &decoder{
		timeout:    1000,
		maxRetries: 2,
	})

	fctx := &fasthttp.RequestCtx{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ctx = http.WithRequestContext(ctx, fctx)
	options := &polaris.Options{
		Timeout:        0,
		MaxRetries:     0,
		Service:        "trpc.service",
		Labels:         []string{"suid", "common.device_id"},
		Namespace:      "Production",
		LimitedRspBody: "429",
		ParseJSONBody:  false,
	}
	ctx, msg := gwmsg.WithNewGWMessage(ctx)
	msg.WithPluginConfig(polaris.PluginName, options)
	fctx.Request.Header.Set("suid", "xxx")
	fctx.Request.SetBodyString(`{"common":{"device_id":"yyy"}}`)
	_, err = limiter.InterceptServer(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	options.ParseJSONBody = true
	_, err = limiter.InterceptServer(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	mockQuota.EXPECT().Get().Return(&model.QuotaResponse{
		Code: model.QuotaResultLimited,
		Info: "",
	})
	_, err = limiter.InterceptServer(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.NotNil(t, err)
}
