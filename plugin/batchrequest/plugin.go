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

// Package batchrequest is a plugin for batch requests
package batchrequest

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/valyala/fasthttp"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	cplugin "trpc.group/trpc-go/trpc-gateway/common/plugin"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
)

const (
	pluginName = "batch_request"
	// Error code for batch request processing
	errBatchRequest = 10004
	// Upstream response code is non-zero, corresponding to HTTP status code 200
	errUpstream    = 10005
	defaultCodeKey = "code"
	defaultMsgKey  = "msg"
)

func init() {
	plugin.Register(pluginName, &Plugin{})
}

// Plugin defines the plugin
type Plugin struct {
}

// Type returns the plugin type
func (p *Plugin) Type() string {
	return cplugin.DefaultType
}

// Setup initializes the plugin
func (p *Plugin) Setup(string, plugin.Decoder) error {
	// Register the plugin
	filter.Register(pluginName, ServerFilter, nil)
	if err := gerrs.Register(errBatchRequest, fasthttp.StatusServiceUnavailable); err != nil {
		return gerrs.Wrap(err, "register batch request code err")
	}
	if err := gerrs.Register(errUpstream, fasthttp.StatusOK); err != nil {
		return gerrs.Wrap(err, "register batch request code err")
	}

	return nil
}

// Options represents the plugin configuration
type Options struct {
	// Name of the status code field, default is "code"; supports nested structure, e.g.: {"common":{"code":0}},
	// then fill in "common.code"
	CodePath string `yaml:"code_path"`
	// Path of the "msg" field, default is "msg"; supports nested structure, e.g.: {"common":{"msg":"success"}},
	// then fill in "common.msg"
	MsgPath string `yaml:"msg_path"`
	// Success status code, default is 0
	SuccessCode int64 `yaml:"success_code"`
	// Success response body
	SuccessBody []byte `yaml:"-"`
	// List of request interfaces
	RequestList []*Request `yaml:"request_list"`
}

// Request represents the request configuration
type Request struct {
	// Name of the request interface
	Method string `yaml:"method"`
	// Path of the target data in the original response body, e.g.: {"data":{"source":{}}}, then fill in "data.source"
	SourceDatePath string `yaml:"source_date_path"`
	// Path where the target data is inserted into the result response body, e.g.: put it into {"data":{"target":{}}},
	// then fill in "data.target"
	TargetDataPath string `yaml:"target_data_path"`
	// Ignore errors
	IgnoreErr bool `yaml:"ignore_err"`
	// Path of the status code in the original response, e.g.: {"common":{"code":0}}, then fill in "common.code",
	// default is "code"
	CodePath string `yaml:"code_path"`
	// Path of the "msg" in the original response, e.g.: {"common":{"msg":"ok"}}, then fill in "common.msg",
	// default is "msg"
	MsgPath string `yaml:"msg_path"`
	// Success status code in the original response, default is 0
	SuccessCode int64 `yaml:"success_code"`
}

// CheckConfig validates the plugin configuration and returns a parsed configuration object with the correct types.
// Used in the ServerFilter method for parsing.
func (p *Plugin) CheckConfig(_ string, decoder plugin.Decoder) error {
	options := &Options{}
	if err := decoder.Decode(options); err != nil {
		return gerrs.Wrap(err, "decode config err")
	}

	if options.CodePath == "" {
		options.CodePath = defaultCodeKey
	}

	if options.MsgPath == "" {
		options.MsgPath = defaultMsgKey
	}

	// Limit the maximum concurrency to 5 to prevent abuse
	if len(options.RequestList) > 5 {
		return errs.Newf(gerrs.ErrWrongConfig, "the maximum number of concurrency is 5")
	}

	tmpMap := make(map[string]struct{})
	for _, request := range options.RequestList {
		if request.Method == "" {
			return errs.Newf(gerrs.ErrWrongConfig, "empty method")
		}
		if request.SourceDatePath == "" || request.TargetDataPath == "" {
			return errs.Newf(gerrs.ErrWrongConfig, "empty data path")
		}
		if _, ok := tmpMap[request.Method]; ok {
			return errs.Newf(gerrs.ErrWrongConfig, "duplicate method:%s", request.Method)
		}
		tmpMap[request.Method] = struct{}{}
		if request.CodePath == "" {
			request.CodePath = defaultCodeKey
		}
		if request.MsgPath == "" {
			request.MsgPath = defaultMsgKey
		}

	}
	// Get the success response body
	successBody, err := sjson.SetBytes([]byte(`{}`), options.CodePath, options.SuccessCode)
	if err != nil {
		return errs.Newf(gerrs.ErrWrongConfig, "set success code err,code path:%s,success code:%v",
			options.CodePath, options.SuccessCode)
	}
	options.SuccessBody = successBody
	return nil
}

// ServerFilter is the server-side interceptor
func ServerFilter(ctx context.Context, _ interface{}, _ filter.ServerHandleFunc) (interface{}, error) {
	// No need to make actual requests
	if err := handle(ctx); err != nil {
		return nil, gerrs.Wrap(err, "batch request err")
	}
	return nil, nil
}

func handle(ctx context.Context) error {
	defer func() {
		if r := recover(); r != nil {
			log.Error(ctx, "batch request panic:%s,stack:%s", r, debug.Stack())
		}
	}()
	// Parse the plugin configuration
	pluginConfig := gwmsg.GwMessage(ctx).PluginConfig(pluginName)
	if pluginConfig == nil {
		log.ErrorContextf(ctx, "get no batch request config")
		return nil
	}
	options, ok := pluginConfig.(*Options)
	if !ok {
		log.ErrorContextf(ctx, "invalid batch request config")
		return nil
	}
	return batchRequest(ctx, options)
}

// batchRequest performs batch requests
func batchRequest(ctx context.Context, options *Options) error {
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return nil
	}

	sm := sync.Map{}
	// Copy the request content and make concurrent requests
	var batchReqFunc []func() error
	for _, request := range options.RequestList {
		nRequest := request
		batchReqFunc = append(batchReqFunc, func() error {
			rspBody, err := requestAndHandle(ctx, nRequest, options)
			if err != nil {
				if rspBody != nil {
					fctx.Request.SetBody(rspBody)
				}
				return gerrs.Wrap(err, "request and handle err")
			}
			sm.Store(nRequest.Method, rspBody)
			return nil
		})
	}
	if err := trpc.GoAndWait(batchReqFunc...); err != nil {
		return gerrs.Wrap(err, "current request err")
	}
	rsp, err := assembleRsp(ctx, options, &sm)
	if err != nil {
		return gerrs.Wrap(err, "assemble rsp err")
	}
	fctx.Response.SetBody(rsp)
	return nil
}

// assembleRsp assembles the response body
func assembleRsp(ctx context.Context, options *Options, sm *sync.Map) ([]byte, error) {
	// Assemble the body and set it in the response body
	var err error
	successBody := options.SuccessBody
	for _, request := range options.RequestList {
		val, ok := sm.LoadAndDelete(request.Method)
		if !ok {
			log.DebugContextf(ctx, "load no rsp body")
			continue
		}
		if val == nil {
			continue
		}
		rspBody, ok := val.([]byte)
		if !ok {
			log.DebugContextf(ctx, "assert rsp body err")
			continue
		}
		sourceData := gjson.GetBytes(rspBody, request.SourceDatePath)
		if !sourceData.Exists() {
			log.DebugContextf(ctx, "data not exist,rsp body:%s,path:%s", string(rspBody), request.SourceDatePath)
			continue
		}
		successBody, err = sjson.SetBytes(successBody, request.TargetDataPath, sourceData.Value())
		if err != nil {
			return nil, errs.Wrap(err, errBatchRequest, "set target data err")
		}
	}
	return successBody, nil
}

// Request and handle errors
func requestAndHandle(ctx context.Context, reqOpts *Request,
	globalOpts *Options) ([]byte, error) {
	rspBody, err := DoRequest(ctx, reqOpts)
	if err != nil && !reqOpts.IgnoreErr {
		return nil, gerrs.Wrap(err, "request err")
	}
	// If the response body is empty, it indicates an exception, return nil directly
	if rspBody == nil {
		return nil, nil
	}
	// Judge the response code
	codeResult := gjson.GetBytes(rspBody, reqOpts.CodePath)
	if !codeResult.Exists() {
		return nil, errs.New(errBatchRequest, "get no code")
	}
	if codeResult.Int() == reqOpts.SuccessCode {
		return rspBody, nil
	}
	// Upstream response failed
	// Ignore the error and return nil directly
	if reqOpts.IgnoreErr {
		return nil, nil
	}

	// Assemble the failed response body
	errBody, err := sjson.SetBytes([]byte(`{}`), globalOpts.CodePath, codeResult.Int())
	if err != nil {
		return nil, errs.Newf(errBatchRequest, "set err code err:%s", err)
	}
	errMsg := gjson.GetBytes(rspBody, reqOpts.MsgPath).String()
	errBody, err = sjson.SetBytes(errBody, globalOpts.MsgPath, errMsg)
	if err != nil {
		return nil, errs.Newf(errBatchRequest, "set err msg err:%s", err)
	}

	return errBody, errs.New(errUpstream, "request err")
}

// FasthttpDo rename the method FasthttpDo
var FasthttpDo = fasthttp.Do

// DoRequest makes a local request
func DoRequest(ctx context.Context, request *Request) ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error(ctx, "do request panic:%s,stack:%s", r, debug.Stack())
		}
	}()
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return nil, nil
	}
	if request == nil || request.Method == "" {
		return nil, errs.New(gerrs.ErrWrongConfig, "invalid batch request config")
	}
	// Get the local IP
	if len(trpc.GlobalConfig().Server.Service) == 0 {
		return nil, errs.New(gerrs.ErrWrongConfig, "invalid global service config")
	}
	host := fmt.Sprintf("%s:%v", trpc.GlobalConfig().Server.Service[0].IP,
		trpc.GlobalConfig().Server.Service[0].Port)
	// Make an HTTP request
	otherCtx := &fasthttp.RequestCtx{}
	otherCtx.Init(&fctx.Request, nil, nil)
	otherCtx.Request.URI().SetHost(host)
	otherCtx.Request.URI().SetPath(request.Method)
	err := FasthttpDo(&otherCtx.Request, &otherCtx.Response)
	if err != nil {
		return nil, errs.Wrap(err, errBatchRequest, "do batch request err")
	}
	if otherCtx.Response.StatusCode() != fasthttp.StatusOK {
		return nil, errs.Newf(errBatchRequest, "http status code:%v", otherCtx.Response.StatusCode())
	}
	log.DebugContextf(ctx, "resp stats:%v, req:%s,body:%s", otherCtx.Response.StatusCode(),
		string(otherCtx.Request.Body()), string(otherCtx.Response.Body()))
	return otherCtx.Response.Body(), nil
}
