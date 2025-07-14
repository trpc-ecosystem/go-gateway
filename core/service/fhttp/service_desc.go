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
	"fmt"
	"strconv"

	"github.com/valyala/fasthttp"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/core/router"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	terrs "trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/metrics"
	"trpc.group/trpc-go/trpc-go/server"
)

// ServiceDesc is the service description.
var ServiceDesc = server.ServiceDesc{
	HandlerType: nil,
	Methods:     []server.Method{generateMethod("*", h.HTTPHandler)},
}

// RegisterFastHTTPService registers the fast HTTP service.
func RegisterFastHTTPService(s server.Service) {
	h.SetRouter(router.GetRouter(ProtocolName))
	_ = s.Register(&ServiceDesc, nil)
}

// generateMethod generates a server method.
func generateMethod(pattern string, handler func(ctx context.Context) error) server.Method {
	handlerFunc := func(svr interface{}, ctx context.Context, _ server.FilterFunc) (rsp interface{}, err error) {
		defer func() {
			if err != nil {
				// Convert the error type to a frame error so that trpc recognizes the error code
				err = terrs.NewFrameError(terrs.Code(err), err.Error())
				DefaultReportErr(ctx, err)
			}
		}()
		handleFunc := func(ctx context.Context, reqBody interface{}) (interface{}, error) {
			reqCtx := http.RequestContext(ctx)
			if reqCtx == nil {
				return nil, errors.New("http Handle missing http header in context")
			}
			return nil, handler(ctx)
		}

		// Request preprocessing
		if err := DefaultPreProcessRoute(ctx); err != nil {
			return nil, gerrs.Wrap(err, "pre process router err")
		}

		// Route matching
		targetService, err := h.router.GetMatchRouter(ctx)
		if err != nil {
			fCtx := http.RequestContext(ctx)
			if terrs.Code(err) != gerrs.ErrPathNotFound {
				log.ErrorContextf(ctx, "get http router failed:%s,path:%s", err, fCtx.Path())
			}
			log.Debugf("get http router failed:%s,path:%s", err, fCtx.Path())
			fCtx.SetStatusCode(fasthttp.StatusNotFound)
			return nil, err
		}
		// Set route information to the context for use in the handleFunc function
		gMsg := gwmsg.GwMessage(ctx)
		gMsg.WithTargetService((*client.BackendConfig)(targetService.BackendConfig))

		// Add all gateway plugin configurations to the context for use in the plugin logic
		var pluginsNameList []string
		for _, plugin := range targetService.Plugins {
			pluginsNameList = append(pluginsNameList, plugin.Name)
			gMsg.WithPluginConfig(plugin.Name, plugin.Props)
		}

		// Execute the plugins and perform the final request forwarding
		rsp, err = filter.ServerChain(targetService.Filters).Filter(ctx, nil, handleFunc)
		if err != nil {
			return rsp, gerrs.Wrap(err, "execute_proxy_err")
		}
		return rsp, nil
	}

	return server.Method{
		Name: pattern,
		Func: handlerFunc,
	}
}

// PreProcessRouteFunc is a function type used for preprocessing before route matching. It can be used to perform
// pre-processing on requests.
type PreProcessRouteFunc func(ctx context.Context) error

// DefaultPreProcessRoute is the default pre-processing function before route matching. It can be overridden by the
// business logic. Note: it will affect the core forwarding logic, so be careful.
var DefaultPreProcessRoute PreProcessRouteFunc = func(ctx context.Context) error {
	return nil
}

// ReportErrFunc is a function type used for error reporting.
type ReportErrFunc func(ctx context.Context, err error)

// DefaultReportErr is the default error reporting function. It can be overridden by the user.
var DefaultReportErr ReportErrFunc = func(ctx context.Context, err error) {
	if err == nil {
		return
	}
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return
	}
	method := codec.Message(ctx).CalleeMethod()
	if terrs.Code(err) == gerrs.ErrPathNotFound || fctx.Response.StatusCode() == fasthttp.StatusNotFound {
		method = "NotFound"
	}
	// Report error code for monitoring and alerting. Error code can be used to differentiate between gateway errors
	// and upstream service errors.
	dims := []*metrics.Dimension{
		{
			Name:  "err_code",
			Value: fmt.Sprintf("%v", terrs.Code(err)),
		},
		{
			Name:  "http_code",
			Value: strconv.Itoa(fctx.Response.StatusCode()),
		},
		{
			Name:  "method",
			Value: method,
		},
	}
	indices := []*metrics.Metrics{
		metrics.NewMetrics("proxy_err_count", float64(1), metrics.PolicySUM),
	}
	err = metrics.Report(metrics.NewMultiDimensionMetricsX(gerrs.GatewayERRKey, dims, indices))
	if err != nil {
		log.Errorf("report proxy err count failed:%s", err)
	}
}
