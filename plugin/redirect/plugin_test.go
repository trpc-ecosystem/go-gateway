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

package redirect_test

import (
	"context"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/plugin"
	"trpc.group/trpc-go/trpc-gateway/plugin/redirect"
)

func TestPlugin_CheckConfig(t *testing.T) {
	p := &redirect.Plugin{}
	_ = p.Setup("", nil)
	opts := &redirect.Options{
		HTTPToHTTPS:       true,
		URI:               "https://127.0.0.1",
		RegexURI:          []string{"^/iresty/(.)/(.)/(.*)", "/$1-$2-$3"},
		RetCode:           0,
		AppendQueryString: false,
	}

	// Configuration validation fails
	decoder := &plugin.PropsDecoder{Props: opts}
	err := p.CheckConfig("", decoder)
	assert.NotNil(t, err)

	// Configuration validation fails
	opts.HTTPToHTTPS = false
	opts.URI = ""
	opts.RegexURI = []string{"^/iresty/(.)/(.)/(.*)"}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)

	opts.RegexURI = nil
	opts.RegexURI = []string{"^/iresty/(.)/(.)/(.*)", "/$1-$2-$3"}
	// Configuration validation succeeds
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)

	// Non-HTTP request
	_, err = redirect.ServerFilter(context.Background(), nil, func(ctx context.Context,
		req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Plugin configuration not obtained
	fctx := &fasthttp.RequestCtx{}
	ctx := http.WithRequestContext(context.Background(), fctx)
	ctx, msg := gwmsg.WithNewGWMessage(ctx)
	_, err = redirect.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Plugin configuration not obtained
	msg.WithPluginConfig("redirect", struct{}{})
	_, err = redirect.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Response success
	opts = &redirect.Options{
		HTTPToHTTPS:       false,
		URI:               "",
		RegexURI:          []string{"^/iresty/(.)/(.)/(.*)", "/$1-$2-$3"},
		RetCode:           0,
		AppendQueryString: true,
	}
	decoder = &plugin.PropsDecoder{Props: opts}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig("redirect", decoder.DecodedProps)
	fctx.Request.URI().SetPath("/iresty/a/b/c")
	fctx.Request.URI().SetQueryString("name=quon")
	_, err = redirect.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Equal(t, "/a-b-c?name=quon", string(fctx.Response.Header.Peek("Location")))
	assert.Nil(t, err)
	fctx.Response.Header.Del("Location")

	// response successful
	opts = &redirect.Options{
		HTTPToHTTPS:       false,
		URI:               "",
		RegexURI:          []string{"^/iresty/(.)/(.)/(.*)", "/a/b/c"},
		RetCode:           0,
		AppendQueryString: true,
	}
	decoder = &plugin.PropsDecoder{Props: opts}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig("redirect", decoder.DecodedProps)
	_, err = redirect.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Equal(t, "/a/b/c?name=quon", string(fctx.Response.Header.Peek("Location")))
	assert.Nil(t, err)
	fctx.Response.Header.Del("Location")

	fctx.Request.URI().SetPath("/test/a/b/c")
	_, err = redirect.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Equal(t, "", string(fctx.Response.Header.Peek("Location")))
	assert.Nil(t, err)

	// HTTPToHTTPS
	opts = &redirect.Options{
		HTTPToHTTPS: true,
	}
	decoder = &plugin.PropsDecoder{Props: opts}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig("redirect", decoder.DecodedProps)
	fctx.Request.URI().SetPath("/iresty/a/b/c")
	fctx.Request.URI().SetHost("view.inews.qq.com")
	_, err = redirect.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Equal(t, "https://view.inews.qq.com/iresty/a/b/c", string(fctx.Response.Header.Peek("Location")))
	assert.Nil(t, err)

	// URI
	opts = &redirect.Options{
		URI: "https://127.0.0.1/",
	}
	decoder = &plugin.PropsDecoder{Props: opts}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig("redirect", decoder.DecodedProps)
	fctx.Request.URI().SetPath("/iresty/a/b/c")
	_, err = redirect.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "https://127.0.0.1/", string(fctx.Response.Header.Peek("Location")))

	// URI
	opts = &redirect.Options{
		URI:               "https://127.0.0.1?a=b",
		AppendQueryString: true,
	}
	decoder = &plugin.PropsDecoder{Props: opts}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig("redirect", decoder.DecodedProps)
	fctx.Request.URI().SetPath("/iresty/a/b/c")
	fctx.Request.URI().SetQueryString("name=quon")
	_, err = redirect.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "https://127.0.0.1?a=b&name=quon", string(fctx.Response.Header.Peek("Location")))
	fctx.Request.URI().SetQueryString("")

	// URI
	opts = &redirect.Options{
		URI:               "https://127.0.0.1",
		AppendQueryString: true,
	}
	decoder = &plugin.PropsDecoder{Props: opts}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig("redirect", decoder.DecodedProps)
	fctx.Request.URI().SetPath("/iresty/a/b/c")
	_, err = redirect.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	fctx.Response.Header.Del("Location")

	// target err
	opts = &redirect.Options{}
	msg.WithPluginConfig("redirect", opts)
	fctx.Request.URI().SetPath("/iresty/a/b/c")
	_, err = redirect.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "", string(fctx.Response.Header.Peek("Location")))
}

func TestRegexp(t *testing.T) {
	r := `^/a/[A-Z]{3}20(03|04|05|06|07|08|09|10|11|12|13|14|15|16|17)\d+`
	e, err := regexp.Compile(r)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(e.MatchString("/a/20220819A00WNO00"))
}
