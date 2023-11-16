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

// Package traceid provides trace ID handling
package traceid

import (
	"context"
	"runtime/debug"

	"go.opentelemetry.io/otel/trace"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	gwplugin "trpc.group/trpc-go/trpc-gateway/common/plugin"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
)

const (
	pluginName    = "traceid"
	headerTraceID = "X-Galileo-Trace-Id"
)

func init() {
	plugin.Register(pluginName, &Plugin{})
}

// Plugin is the traceid plugin definition
type Plugin struct{}

// Type returns the traceid plugin type
func (p *Plugin) Type() string {
	return gwplugin.DefaultType
}

// Setup initializes the traceid plugin
func (p *Plugin) Setup(string, plugin.Decoder) error {
	// Register the plugin
	filter.Register(pluginName, ServerFilter, nil)
	return nil
}

// Options is the plugin configuration
type Options struct{}

// CheckConfig validates the traceid plugin configuration and returns the parsed configuration object for use in the
// ServerFilter method
func (p *Plugin) CheckConfig(_ string, decoder plugin.Decoder) error {
	options := &Options{}
	if err := decoder.Decode(options); err != nil {
		return gerrs.Wrap(err, "decode traceid config error")
	}
	return nil
}

// ServerFilter is the server-side interceptor
func ServerFilter(ctx context.Context, req interface{}, handler filter.ServerHandleFunc) (interface{}, error) {
	rsp, err := handler(ctx, req)
	setTraceID(ctx)
	return rsp, err
}

// setTraceID adds the trace ID to the response header
func setTraceID(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Error(ctx, "set traceid panic: %s, stack: %s", r, debug.Stack())
		}
	}()
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return
	}
	span := trace.SpanContextFromContext(ctx)
	// If the filter is not in the span or the span is not sampled, do not set the trace ID
	if !span.IsValid() || !span.IsSampled() {
		return
	}

	fctx.Response.Header.Set(headerTraceID, span.TraceID().String())
}
