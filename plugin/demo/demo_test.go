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

package demo

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/plugin"
	mockplugin "trpc.group/trpc-go/trpc-gateway/plugin/mock"
)

func TestAuthPlugin_Setup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	p := &Plugin{}
	assert.Nil(t, p.Setup("", nil))

	assert.NotNil(t, p.Setup("", nil))

	p.DependsOn()
	demoConfig := &Options{}
	decoder := &plugin.PropsDecoder{Props: demoConfig}
	assert.Nil(t, p.CheckConfig("", decoder))

	mockDecoder := mockplugin.NewMockDecoder(ctrl)
	mockDecoder.EXPECT().Decode(gomock.Any()).Return(errors.New("err"))
	assert.NotNil(t, p.CheckConfig("", mockDecoder))
}

func TestServerFilter(t *testing.T) {
	_, err := ServerFilter(context.Background(), nil,
		func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
			return nil, nil
		})
	assert.NotNil(t, err)

	ctx, msg := gwmsg.WithNewGWMessage(context.Background())
	msg.WithPluginConfig(pluginName, "invalid option")
	fctx := &fasthttp.RequestCtx{}
	ctx = http.WithRequestContext(ctx, fctx)
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.NotNil(t, err)

	msg.WithPluginConfig(pluginName, &Options{})
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, errors.New("err")
	})
	assert.NotNil(t, err)

	msg.WithPluginConfig(pluginName, &Options{})
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
}
