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

package traceid_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"go.opentelemetry.io/otel/trace"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/plugin"
	"trpc.group/trpc-go/trpc-gateway/plugin/traceid"
)

func TestPlugin_CheckConfig(t *testing.T) {
	p := &traceid.Plugin{}
	_ = p.Setup("", nil)
	opts := &traceid.Options{}

	decoder := &plugin.PropsDecoder{Props: opts}
	// Configuration validation success
	err := p.CheckConfig("", decoder)
	assert.Nil(t, err)

	// Non-HTTP request
	_, err = traceid.ServerFilter(context.Background(), nil, func(ctx context.Context,
		req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Set up the context
	fctx := &fasthttp.RequestCtx{}
	ctx := http.WithRequestContext(context.Background(), fctx)
	ctx, _ = gwmsg.WithNewGWMessage(ctx)

	// No span ID
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: [16]byte{0x01},
		//SpanID:     [8]byte{0x01},
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	ctx = trace.ContextWithSpanContext(ctx, sc)
	_, err = traceid.ServerFilter(ctx, nil, func(ctx context.Context,
		req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "", string(fctx.Response.Header.Peek("X-Galileo-Trace-Id")))

	// No sampling
	sc = trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: [16]byte{0x01},
		SpanID:  [8]byte{0x01},
		//TraceFlags: trace.FlagsSampled,
		Remote: true,
	})
	ctx = trace.ContextWithSpanContext(ctx, sc)
	_, err = traceid.ServerFilter(ctx, nil, func(ctx context.Context,
		req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "", string(fctx.Response.Header.Peek("X-Galileo-Trace-Id")))

	// Set up the context with valid span ID and sampling
	sc = trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    [16]byte{0x01},
		SpanID:     [8]byte{0x01},
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	ctx = trace.ContextWithSpanContext(ctx, sc)
	_, err = traceid.ServerFilter(ctx, nil, func(ctx context.Context,
		req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "01000000000000000000000000000000",
		string(fctx.Response.Header.Peek("X-Galileo-Trace-Id")))
}
