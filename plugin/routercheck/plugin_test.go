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

package routercheck_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/plugin"
	"trpc.group/trpc-go/trpc-gateway/plugin/routercheck"
)

func TestPlugin_CheckConfig(t *testing.T) {
	p := &routercheck.Plugin{}
	_ = p.Setup("", nil)
	opts := &routercheck.Options{}

	// check config failed
	decoder := &plugin.PropsDecoder{Props: opts}
	err := p.CheckConfig("", decoder)
	assert.Nil(t, err)

	// not http request
	_, err = routercheck.ServerFilter(context.Background(), nil, func(ctx context.Context,
		req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// check success
	fctx := &fasthttp.RequestCtx{}
	ctx := http.WithRequestContext(context.Background(), fctx)
	ctx, _ = gwmsg.WithNewGWMessage(ctx)
	fctx.Request.SetBody([]byte(`
router: # Route configuration
  - method: ^/v1/user/ # Regex route
    is_regexp: true  # Whether it is a regex route, set to true to perform regex matching
    id: "path:^/v1/user/" # Route ID, used to identify a route for debugging (method will be duplicated)
    rewrite: /v1/user/info # Rewrite path
    target_service: # Upstream services
      - service: trpc.user.service # Service name, corresponding to the name in the client configuration
        weight: 10 # Service weight, the sum of weights cannot be 0
client: # Upstream service configuration, consistent with the trpc protocol
  - name: trpc.user.service
    namespace: Development
    network: tcp
    target: xxxx
    protocol: fasthttp`))
	_, err = routercheck.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	codeResult := gjson.GetBytes(fctx.Response.Body(), "code")
	assert.True(t, codeResult.Exists())
	assert.Equal(t, int64(0), codeResult.Int())

	// check failed
	fctx.Request.SetBody([]byte(`
router: # Route configuration
  - method: ^/v1/user/ # Regex route
    is_regexp: true  # Whether it is a regex route, set to true to perform regex matching
    id: "path:^/v1/user/" # Route ID, used to identify a route for debugging (method will be duplicated)
    rewrite: /v1/user/info # Rewrite path
    target_service: # Upstream services
      - service: trpc.user.service # Service name, corresponding to the name in the client configuration
        weight: 10 # Service weight, the sum of weights cannot be 0
client: # Upstream service configuration, consistent with the trpc protocol
  - name: trpc.user.service
    namespace: Development
    network: tcp
    target: xxxx
    protocolxxx: fasthttp`))
	_, err = routercheck.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	codeResult = gjson.GetBytes(fctx.Response.Body(), "code")
	assert.True(t, codeResult.Exists())
	assert.NotEqual(t, int64(0), codeResult.Int())

	// yaml parse error
	fctx.Request.SetBody([]byte(`
router: # Route configuration
  - method: ^/v1/user/ # Regex route
    is_regexp: true  # Whether it is a regex route, set to true to perform regex matching
    id: "path:^/v1/user/" # Route ID, used to identify a route for debugging (method will be duplicated)
    rewrite: /v1/user/info # Rewrite path
    target_service: # Upstream services
      - service: trpc.user.service # Service name, corresponding to the name in the client configuration
        weight: 10 # Service weight, the sum of weights cannot be 0
client: # Upstream service configuration, consistent with the trpc protocol
  - name: trpc.user.service
    namespace: Development
    network: tcp
    target: xxxx
    protocolxxx: fasthttp`))
	_, err = routercheck.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	codeResult = gjson.GetBytes(fctx.Response.Body(), "code")
	assert.True(t, codeResult.Exists())
	assert.NotEqual(t, int64(0), codeResult.Int())
}
