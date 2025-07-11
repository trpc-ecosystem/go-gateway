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

package accesslog_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/plugin"
	"trpc.group/trpc-go/trpc-gateway/plugin/accesslog"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	tplugin "trpc.group/trpc-go/trpc-go/plugin"
)

func TestPlugin_CheckConfig(t *testing.T) {
	p := &accesslog.Plugin{}
	tDecoder := &tplugin.YamlNodeDecoder{
		Node: &yaml.Node{},
	}
	err := p.Setup("", tDecoder)
	assert.Nil(t, err)
	opts := &accesslog.Options{
		FieldList: []map[string]string{
			{"suid": "suid"},
		},
	}

	// check config	success
	decoder := &plugin.PropsDecoder{Props: opts}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)

	// execute filter success
	_, err = accesslog.ServerFilter(context.Background(), nil, func(ctx context.Context,
		req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	fctx := &fasthttp.RequestCtx{}
	ctx := http.WithRequestContext(context.Background(), fctx)
	ctx, msg := gwmsg.WithNewGWMessage(ctx)
	msg.WithTargetService(&client.BackendConfig{})

	msg.WithPluginConfig("accesslog", nil)
	_, err = accesslog.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, errors.New("err")
	})
	assert.NotNil(t, err)

	msg.WithTargetService(nil)
	msg.WithPluginConfig("accesslog", nil)
	_, err = accesslog.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	msg.WithPluginConfig("accesslog", "nil")
	_, err = accesslog.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	msg.WithPluginConfig("accesslog", decoder.DecodedProps)
	_, err = accesslog.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	msg.WithPluginConfig("accesslog", decoder.DecodedProps)
	_, err = accesslog.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	msg.WithPluginConfig("accesslog", opts)
	_, err = accesslog.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	msg.WithPluginConfig("accesslog", opts)
	_, err = accesslog.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	fctx.Request.URI().SetPath("/")
	ctx, tmsg := codec.WithNewMessage(ctx)
	tmsg.WithCallerMethod("/")
	msg.WithPluginConfig("accesslog", opts)
	_, err = accesslog.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
}
