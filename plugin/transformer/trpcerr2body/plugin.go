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

// Package trpcerr2body converts trpc errors to response bodies
package trpcerr2body

import (
	"context"
	"runtime/debug"
	"strconv"

	"github.com/tidwall/sjson"
	"github.com/valyala/fasthttp"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	cplugin "trpc.group/trpc-go/trpc-gateway/common/plugin"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
)

const (
	pluginName = "trpcerr2body"
	// Error converting err to body
	errErr2Body = 10008
)

func init() {
	plugin.Register(pluginName, &Plugin{})
}

// Plugin defines the plugin
type Plugin struct {
}

// Type gets the plugin type
func (p *Plugin) Type() string {
	return cplugin.DefaultType
}

// Setup initializes the plugin
func (p *Plugin) Setup(string, plugin.Decoder) error {
	// Register the plugin
	filter.Register(pluginName, ServerFilter, nil)
	if err := gerrs.Register(errErr2Body, fasthttp.StatusInternalServerError); err != nil {
		return gerrs.Wrap(err, "register trpcerr2body plugin code err")
	}
	return nil
}

// Options defines the plugin options
type Options struct {
	// Error code path, default is "code", supports multiple levels, e.g., common.code
	CodePath string `yaml:"code_path"`
	// Data type for code, "number" for int, "string" for string, default is "number"
	CodeValType ValType `yaml:"code_val_type"`
	// Error message path, default is "msg", supports multiple levels, e.g., common.msg
	MsgPath string `yaml:"msg_path"`
}

// ValType represents the value type
type ValType string

const (
	// ValStr represents the string type
	ValStr ValType = "string"
	// ValNumber represents the int type
	ValNumber ValType = "number"
)

// CheckConfig validates the plugin configuration and returns a parsed configuration object with the correct types.
// Used in the ServerFilter method for parsing.
func (p *Plugin) CheckConfig(_ string, decoder plugin.Decoder) error {
	options := &Options{}
	if err := decoder.Decode(options); err != nil {
		return gerrs.Wrap(err, "decode trpcerr2body config err")
	}
	if options.CodePath == "" {
		options.CodePath = "code"
	}
	if options.CodeValType == "" {
		options.CodeValType = ValNumber
	}
	if options.MsgPath == "" {
		options.MsgPath = "msg"
	}
	return nil
}

// ServerFilter is the server interceptor
func ServerFilter(ctx context.Context, req interface{}, handler filter.ServerHandleFunc) (interface{}, error) {
	rsp, berr := handler(ctx, req)
	// Put error information into the body
	if e := err2body(ctx, berr); e != nil {
		// Degrade error conversion to prevent affecting normal requests
		log.ErrorContextf(ctx, "err2body err:%s", e)
	}
	if berr != nil {
		return nil, berr
	}
	return rsp, nil
}

// Override the forwarded environment
func err2body(ctx context.Context, berr error) error {
	defer func() {
		if r := recover(); r != nil {
			log.Error(ctx, "err2body handle panic:%s,stack:%s", r, debug.Stack())
		}
	}()

	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return nil
	}
	// Parse plugin configuration
	pluginConfig := gwmsg.GwMessage(ctx).PluginConfig(pluginName)
	if pluginConfig == nil {
		return errs.New(gerrs.ErrWrongConfig, "get no err2body config")
	}
	options, ok := pluginConfig.(*Options)
	if !ok {
		return errs.New(gerrs.ErrWrongConfig, "invalid trpcerr2body config")
	}
	errCode, errMsg := getCodeAndMsg(fctx, berr)
	if len(errCode) == 0 {
		return nil
	}

	// Get the response body
	body := fctx.Response.Body()
	if len(body) == 0 {
		body = []byte("{}")
	}
	var err error
	codeVal, err := getCodeVal(errCode, options.CodeValType)
	if err != nil {
		return gerrs.Wrap(err, "get code val err")
	}
	body, err = sjson.SetBytes(body, options.CodePath, codeVal)
	if err != nil {
		return errs.Wrap(err, errErr2Body, "set code err")
	}
	body, err = sjson.SetBytes(body, options.MsgPath, errMsg)
	if err != nil {
		return errs.Wrap(err, errErr2Body, "set msg err")
	}
	fctx.Response.SetBody(body)
	fctx.Response.Header.SetContentType("application/json")
	return nil
}

// Get the error code and error message
func getCodeAndMsg(fctx *fasthttp.RequestCtx, berr error) (string, string) {
	errMsg := fctx.Response.Header.Peek(thttp.TrpcErrorMessage)
	errCode := fctx.Response.Header.Peek(thttp.TrpcFrameworkErrorCode)
	if len(errCode) != 0 {
		return string(errCode), string(errMsg)
	}
	errCode = fctx.Response.Header.Peek(thttp.TrpcUserFuncErrorCode)
	if len(errCode) != 0 {
		return string(errCode), string(errMsg)
	}
	bterr, ok := gerrs.UnWrap(berr)
	if ok && bterr.Code != 0 {
		return strconv.Itoa(int(bterr.Code)), bterr.Msg
	}
	return "", ""
}

// getCodeVal converts the code value type
func getCodeVal(codeStr string, valType ValType) (interface{}, error) {
	if valType == ValStr {
		return codeStr, nil
	}
	code, err := strconv.Atoi(codeStr)
	if err != nil {
		return code, errs.Wrap(err, errErr2Body, "convert code type err")
	}
	return int32(code), nil
}
