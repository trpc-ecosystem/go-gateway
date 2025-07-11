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

// Package accesslog contains proxy request logging.
package accesslog

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/valyala/fasthttp"
	"go.opentelemetry.io/otel/trace"
	"trpc.group/trpc-go/trpc-gateway/common/convert"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	cplugin "trpc.group/trpc-go/trpc-gateway/common/plugin"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/plugin"
)

const pluginName = "accesslog"

func init() {
	plugin.Register(pluginName, log.DefaultLogFactory)
	plugin.Register(pluginName, &Plugin{})
}

// Plugin defines the plugin.
type Plugin struct {
}

// Type returns the plugin type.
func (p *Plugin) Type() string {
	return cplugin.DefaultType
}

// Setup initializes the plugin.
func (p *Plugin) Setup(string, plugin.Decoder) error {
	// Register the plugin
	filter.Register(pluginName, ServerFilter, nil)
	return nil
}

// Options represents the plugin configuration.
type Options struct {
	// TODO 如何保持有序？
	FieldList []map[string]string `yaml:"field_list"`
}

// CheckConfig validates the plugin configuration and returns the parsed configuration object. Used in the ServerFilter
// method for parsing.
func (p *Plugin) CheckConfig(_ string, decoder plugin.Decoder) error {
	options := &Options{}
	if err := decoder.Decode(options); err != nil {
		return gerrs.Wrap(err, "decode access log config error")
	}
	return nil
}

// ServerFilter represents the proxy information.
func ServerFilter(ctx context.Context, req interface{}, handler filter.ServerHandleFunc) (interface{}, error) {
	node := &registry.Node{}
	gwmsg.GwMessage(ctx).WithTRPCClientOpts([]client.Option{client.WithSelectorNode(node)})
	rsp, err := handler(ctx, req)
	accessLog(ctx, err, node)
	return rsp, err
}

func accessLog(ctx context.Context, err error, node *registry.Node) {
	defer func() {
		if r := recover(); r != nil {
			log.ErrorContextf(ctx, "access_log_panic:%s,stack:%s", r, debug.Stack())
		}
	}()

	// If it is an HTTP proxy, try to get the common parameters for reporting
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return
	}

	fieldList := []log.Field{
		// Interface path, corresponding to the "method" field configured in router.yaml
		{Key: "path", Value: codec.Message(ctx).CallerMethod()},
		// Full interface path, not empty when the interface has rewriting
		{Key: "upstream_path", Value: getUpstreamPath(ctx)},
		// Router ID
		{Key: "router_id", Value: gwmsg.GwMessage(ctx).RouterID()},
		{Key: "err_no", Value: fmt.Sprint(errs.Code(err))},
		{Key: "err_msg", Value: getErrMSG(err)},
		{Key: "local_ip", Value: trpc.GlobalConfig().Global.LocalIP},
		{Key: "upstream_service", Value: node.ServiceName},
		{Key: "upstream_protocol", Value: getUpstreamProtocol(ctx)},
		// Backend service ip:port
		{Key: "upstream_addr", Value: node.Address},
		{Key: "upstream_status", Value: fmt.Sprint(fctx.Response.StatusCode())},
		{Key: "upstream_response_time", Value: node.CostTime.Milliseconds()},
		// Client IP
		{Key: "remote_addr", Value: getClientIPFromContext(fctx)},
		// Trace ID, not empty when using Galileo or Tianjige
		{Key: "traceid", Value: getTraceID(ctx)},
		{Key: "user_agent", Value: string(fctx.Request.Header.UserAgent())},
		{Key: "host", Value: string(fctx.Host())},
		{Key: "referer", Value: string(fctx.Referer())},
		{Key: "server_protocol", Value: string(fctx.Request.Header.Protocol())},
	}

	extFieldList, err := getExtFields(ctx)
	if err != nil {
		log.ErrorContextf(ctx, "get ext accesslog field err:%s", err)
	}
	fieldList = append(fieldList, extFieldList...)
	// Get custom business fields
	fieldList = append(fieldList, DefaultBusinessFields(ctx, node)...)
	getLogger().With(fieldList...).Info("accesslog")
	return
}

// BusinessFields sets custom business log fields
type BusinessFields func(ctx context.Context, node *registry.Node) []log.Field

// DefaultBusinessFields overrides this method to set custom business log fields
var DefaultBusinessFields BusinessFields = func(ctx context.Context, _ *registry.Node) []log.Field {
	return nil
}

// Get extension fields
func getExtFields(ctx context.Context) ([]log.Field, error) {
	fctx := http.RequestContext(ctx)
	// Parse plugin configuration
	pluginConfig := gwmsg.GwMessage(ctx).PluginConfig(pluginName)
	if pluginConfig == nil {
		return nil, errs.New(gerrs.ErrWrongConfig, "get no accesslog config")
	}
	options, ok := pluginConfig.(*Options)
	if !ok {
		return nil, errs.New(gerrs.ErrWrongConfig, "invalid accesslog config")
	}
	log.DebugContextf(ctx, "accesslog_config:%s", convert.ToJSONStr(options))
	// Iterate through the configuration to get business parameters
	var fieldList []log.Field
	for _, fieldMap := range options.FieldList {
		for key, fieldName := range fieldMap {
			fieldList = append(fieldList,
				log.Field{
					Key:   key,
					Value: http.GetString(fctx, fieldName),
				},
			)
		}
	}
	return fieldList, nil
}

// Get the protocol
func getUpstreamProtocol(ctx context.Context) string {
	if gwmsg.GwMessage(ctx).TargetService() == nil {
		return ""
	}
	return gwmsg.GwMessage(ctx).TargetService().Protocol
}

// Get the full interface path
func getUpstreamPath(ctx context.Context) string {
	fctx := http.RequestContext(ctx)
	// Add reporting for the full path. For example: /a/{article_id}, msg.CallerMethod() is /a/, need to add reporting for
	// /a/{article_id} for troubleshooting purposes
	if string(fctx.Path()) != codec.Message(ctx).CallerMethod() {
		return string(fctx.Path())
	}
	return ""
}

// Get the error code
func getErrMSG(err error) string {
	if err == nil {
		return ""
	}
	return errs.Msg(err)
}

// Get the trace ID
func getTraceID(ctx context.Context) string {
	span := trace.SpanContextFromContext(ctx)
	if span.IsValid() {
		return span.TraceID().String()
	}
	return ""
}

// Get the logger
func getLogger() log.Logger {
	logger := log.GetDefaultLogger()
	if l := log.Get(pluginName); l != nil {
		logger = l
	}
	return logger
}

const localAddress = "127.0.0.1"

// GetClientIPFromContext retrieves the user's IP address from the context
func getClientIPFromContext(fCtx *fasthttp.RequestCtx) string {
	clientIPByte := fCtx.Request.Header.Peek(fasthttp.HeaderXForwardedFor)
	clientIPs := strings.Split(string(clientIPByte), ",")
	if len(clientIPs) == 0 {
		return ""
	}

	clientIP := clientIPs[0]
	if len(clientIP) > 0 && clientIP != localAddress {
		return clientIP
	}
	clientIP = strings.TrimSpace(string(fCtx.Request.Header.Peek("X-Real-Ip")))
	if clientIP != "" && clientIP != localAddress {
		return clientIP
	}
	return ""
}
