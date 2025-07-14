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

package cors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/plugin"
)

func TestPlugin_CheckConfig(t *testing.T) {
	p := &Plugin{}
	_ = p.Setup("", nil)
	corsConfig := &Options{
		AllowOrigins:     nil,
		AllowMethods:     nil,
		AllowHeaders:     nil,
		AllowCredentials: false,
		ExposeHeaders:    nil,
		MaxAge:           1000,
	}

	// Failed configuration validation
	corsConfig.AllowCredentials = true
	decoder := &plugin.PropsDecoder{Props: corsConfig}
	err := p.CheckConfig("", decoder)
	assert.NotNil(t, err)

	// Successful configuration validation
	corsConfig.AllowCredentials = false
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)

	// Cross-origin settings success
	fctx := &fasthttp.RequestCtx{}
	ctx := http.WithRequestContext(context.Background(), fctx)
	ctx, msg := gwmsg.WithNewGWMessage(ctx)
	msg.WithPluginConfig("cors", corsConfig)

	fctx.Request.Header.Set(origin, "http://r.inews.qq.com")
	corsConfig.AllowOrigins = []string{"r.inews.qq.com"}
	corsConfig.AllowCredentials = true
	corsConfig.ExposeHeaders = []string{"my-expose-header"}
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// OPTIONS request
	fctx.Request.Header.SetMethod(fasthttp.MethodOptions)
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Custom request headers
	corsConfig.AllowHeaders = []string{"my-customer-header"}
	fctx.Request.Header.SetMethod(fasthttp.MethodOptions)
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Not an HTTP request
	tmpCtx := http.WithRequestContext(ctx, nil)
	_, err = ServerFilter(tmpCtx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	http.WithRequestContext(ctx, fctx)

	// Not a cross-origin request
	fctx.Request.Header.Del(origin)
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	fctx.Request.Header.Set(origin, "http://r.inews.qq.com")

	corsConfig.AllowOrigins = nil
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	corsConfig.AllowOrigins = []string{"om.qq.com"}
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.NotNil(t, err)

	corsConfig.AllowOrigins = []string{"qq.com"}
	fctx.Request.Header.Set(origin, "http://r.inews.qq.com/----")
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
}

func Test_isAllowCORS(t *testing.T) {
	configDomains := []string{"qq.com"}
	originHost := "r.inews.qq.com"
	allow := isAllowCORS(configDomains, originHost)
	assert.True(t, allow)

	configDomains = []string{}
	allow = isAllowCORS(configDomains, originHost)
	assert.False(t, allow)

	configDomains = []string{"r.inews.qq.com"}
	allow = isAllowCORS(configDomains, originHost)
	assert.True(t, allow)
}
