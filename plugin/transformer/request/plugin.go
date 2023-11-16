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

// Package request request transformer plugin
package request

import (
	"bytes"
	"context"
	"runtime/debug"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/convert"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	cplugin "trpc.group/trpc-go/trpc-gateway/common/plugin"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
)

const (
	// Request body transformation
	pluginName = "request_transformer"
	// DelAllKey is a placeholder for deleting all operations
	DelAllKey = "-1"
)

var (
	strPostArgsContentType = []byte("application/x-www-form-urlencoded")
	strMultipartFormData   = []byte("multipart/form-data")
	strJSONContentType     = []byte("application/json")
)

func init() {
	plugin.Register(pluginName, &Plugin{})
}

// Plugin implements the request transformer plugin
type Plugin struct {
}

// Type returns the request transformer plugin type
func (p *Plugin) Type() string {
	return cplugin.DefaultType
}

// Setup initializes the request transformer instance
func (p *Plugin) Setup(string, plugin.Decoder) error {
	filter.Register(pluginName, ServerFilter, nil)
	return nil
}

// KV represents a key-value structure
type KV struct {
	Key string
	Val string
}

// Options represents the parameter options
type Options struct {
	// Rewrite host
	RewriteHost string `yaml:"rewrite_host" json:"rewrite_host"`
	// Delete operations
	// List of reserved headers, if configured, remove operations will not be executed. -1 indicates deleting all
	ReserveHeaders []string `yaml:"reserve_headers" json:"reserve_headers"`
	// Remove headers
	RemoveHeaders []string `yaml:"remove_headers" json:"remove_headers"`
	// List of reserved query parameters, if configured, remove operations will not be executed. -1 indicates deleting all
	ReserveQueryStr []string `yaml:"reserve_query_str" json:"reserve_query_str"`
	RemoveQueryStr  []string `yaml:"remove_query_str" json:"remove_query_str"`
	// List of reserved body parameters, if configured, remove operations will not be executed. -1 indicates deleting all
	ReserveBody []string `yaml:"reserve_body" json:"reserve_body"`
	RemoveBody  []string `yaml:"remove_body"`

	// Rename header configuration fields in the format key:val
	// Renamed headers will not be formatted
	RenameHeaders []string `yaml:"rename_headers" json:"rename_headers"`
	// Parsed key-value pairs
	RenameHeadersKV []*KV `yaml:"-" json:"-"`
	// Rename query parameters in the format key:val
	RenameQueryStr   []string `yaml:"rename_query_str" json:"rename_query_str"`
	RenameQueryStrKV []*KV    `yaml:"-" json:"-"`

	RenameQueryBodyKV []*KV `yaml:"-" json:"-"`

	// Rename body parameters in the format key:val
	RenameBody   []string `yaml:"rename_body" json:"rename_body"`
	RenameBodyKV []*KV    `yaml:"-" json:"-"`

	// Add headers in the format key:val
	AddHeaders   []string `yaml:"add_headers" json:"add_headers"`
	AddHeadersKV []*KV    `yaml:"-" json:"-"`
	// Add query parameters in the format key:val
	AddQueryStr    []string `yaml:"add_query_str" json:"add_query_str"`
	AddQueryStrKV  []*KV    `yaml:"-" json:"-"`
	AddQueryBodyKV []*KV    `yaml:"-" json:"-"`

	// Add body parameters in the format key:val. It will be appended to different request bodies based on
	// the content type.
	AddBody   []string `yaml:"add_body" json:"add_body"`
	AddBodyKV []*KV    `yaml:"-" json:"-"`
}

// CheckConfig validates the plugin configuration and returns the parsed configuration object. Used in the ServerFilter
// method for parsing.
func (p *Plugin) CheckConfig(name string, decoder plugin.Decoder) error {
	options := &Options{}
	if err := decoder.Decode(options); err != nil {
		return gerrs.Wrap(err, "decode request transformer config error")
	}
	log.Infof("plugin %s config: %s", name, convert.ToJSONStr(options))

	// Parse headers
	var err error
	if len(options.RenameHeaders) != 0 {
		options.RenameHeadersKV, err = getKV(options.RenameHeaders)
		if err != nil {
			return gerrs.Wrap(err, "rename header config error")
		}
	}
	// Parse query parameters
	if len(options.RenameQueryStr) != 0 {
		options.RenameQueryStrKV, err = getKV(options.RenameQueryStr)
		if err != nil {
			return gerrs.Wrap(err, "rename query string config error")
		}
	}

	// Parse rename body parameters
	if len(options.RenameBody) != 0 {
		options.RenameBodyKV, err = getKV(options.RenameBody)
		if err != nil {
			return gerrs.Wrap(err, "rename body config error")
		}
	}

	// Parse header parameters
	if len(options.AddHeaders) != 0 {
		options.AddHeadersKV, err = getKV(options.AddHeaders)
		if err != nil {
			return gerrs.Wrap(err, "add headers config error")
		}
	}

	// Parse query parameters
	if len(options.AddQueryStr) != 0 {
		options.AddQueryStrKV, err = getKV(options.AddQueryStr)
		if err != nil {
			return gerrs.Wrap(err, "add query string config error")
		}
	}

	// Parse configuration
	if len(options.AddBody) != 0 {
		options.AddBodyKV, err = getKV(options.AddBody)
		if err != nil {
			return gerrs.Wrap(err, "add body config error")
		}
	}
	return nil
}

// getKV retrieves the key-value configuration
func getKV(list []string) ([]*KV, error) {
	var kvList []*KV
	for _, v := range list {
		if v == "" {
			continue
		}
		arr := strings.Split(v, ":")
		if len(arr) < 2 {
			return nil, errs.New(gerrs.ErrWrongConfig, "invalid kv config")
		}
		if arr[0] == "" {
			continue
		}
		kvList = append(kvList, &KV{
			Key: arr[0],
			Val: arr[1],
		})
	}
	return kvList, nil
}

// ServerFilter sets up server-side CORS validation
func ServerFilter(ctx context.Context, req interface{}, handler filter.ServerHandleFunc) (interface{}, error) {
	if err := transform(ctx); err != nil {
		return nil, gerrs.Wrap(err, "request_transform_err")
	}
	return handler(ctx, req)
}

// Request transformation
func transform(ctx context.Context) error {
	defer func() {
		if r := recover(); r != nil {
			log.ErrorContextf(ctx, "request transformer handle panic: %s", string(debug.Stack()))
		}
	}()
	// Parse plugin configuration
	pluginConfig := gwmsg.GwMessage(ctx).PluginConfig(pluginName)
	if pluginConfig == nil {
		return errs.New(gerrs.ErrPluginConfigNotFound, "no request transformer plugin config found")
	}
	options, ok := pluginConfig.(*Options)
	if !ok {
		return errs.New(gerrs.ErrPluginConfigNotFound, "invalid request transformer plugin config type")
	}
	// Modify the request content
	modifyRequest(ctx, options)
	return nil
}

// Modify the request content
func modifyRequest(ctx context.Context, options *Options) {
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		// Not an HTTP request, do not process
		return
	}
	// Modify request parameters
	if options.RewriteHost != "" {
		fctx.Request.SetHost(options.RewriteHost)
	}
	var modifyBody bool
	// Delete operation
	if deleteOption(ctx, &fctx.Request, options) {
		modifyBody = true
	}
	// Rename operation
	if renameOption(ctx, &fctx.Request, options) {
		modifyBody = true
	}
	if addOption(ctx, &fctx.Request, options) {
		modifyBody = true
	}
	// If the request body has been modified, reset it
	if modifyBody {
		if bytes.HasPrefix(fctx.Request.Header.ContentType(), strPostArgsContentType) {
			fctx.Request.SetBody(fctx.Request.PostArgs().QueryString())
			return
		}
		if bytes.HasPrefix(fctx.Request.Header.ContentType(), strMultipartFormData) {
			body, err := getMultipartFormBody(ctx, fctx)
			if err != nil {
				log.ErrorContextf(ctx, "get multipart form body error: %s", err)
				return
			}
			fctx.Request.SetBody(body)
			return
		}
	}
}

// Convert form to request body
func getMultipartFormBody(ctx context.Context, fctx *fasthttp.RequestCtx) ([]byte, error) {
	form, err := fctx.Request.MultipartForm()
	if err != nil {
		log.ErrorContextf(ctx, "get multipart form error: %s", err)
		return nil, err
	}
	var buf bytebufferpool.ByteBuffer
	if err := fasthttp.WriteMultipartForm(&buf, form, string(fctx.Request.Header.MultipartFormBoundary())); err != nil {
		log.ErrorContextf(ctx, "write multipart form error: %s", err)
		return nil, err
	}
	return buf.B, nil
}

// Add operation or replace
func addOption(ctx context.Context, request *fasthttp.Request, options *Options) bool {
	// Add headers
	if len(options.AddHeadersKV) != 0 {
		for _, kv := range options.AddHeadersKV {
			request.Header.Set(kv.Key, kv.Val)
		}
	}
	// Add query parameters
	if len(options.AddQueryStrKV) != 0 {
		for _, kv := range options.AddQueryStrKV {
			request.URI().QueryArgs().Set(kv.Key, kv.Val)
		}
	}
	var modifyBody bool
	// Add body parameters
	if len(options.AddBodyKV) != 0 {
		for _, kv := range options.AddBodyKV {
			addBody(ctx, request, kv)
		}
		modifyBody = true
	}
	return modifyBody
}

func addBody(ctx context.Context, request *fasthttp.Request, kv *KV) {
	// Handle different content types
	if bytes.HasPrefix(request.Header.ContentType(), strPostArgsContentType) {
		request.PostArgs().Set(kv.Key, kv.Val)
		return
	}
	if bytes.HasPrefix(request.Header.ContentType(), strJSONContentType) {
		newBody, _ := sjson.SetBytes(request.Body(), kv.Key, kv.Val)
		request.SetBody(newBody)
		return
	}
	if bytes.HasPrefix(request.Header.ContentType(), strMultipartFormData) {
		form, err := request.MultipartForm()
		if err != nil {
			log.ErrorContextf(ctx, "get multipart error: %s", err)
			return
		}
		form.Value[kv.Key] = []string{kv.Val}
		return
	}
	// Do nothing for other content types
	log.WarnContextf(ctx, "unsupported content-type: %s", request.Header.ContentType())
	return
}

// Rename operation
func renameOption(ctx context.Context, request *fasthttp.Request, options *Options) bool {
	// Rename headers
	if len(options.RenameHeadersKV) != 0 {
		for _, kv := range options.RenameHeadersKV {
			val := request.Header.Peek(kv.Key)
			request.Header.DisableNormalizing()
			request.Header.SetBytesV(kv.Val, val)
			request.Header.EnableNormalizing()
			request.Header.Del(kv.Key)
		}
	}
	// Rename query parameters
	if len(options.RenameQueryStrKV) != 0 {
		for _, kv := range options.RenameQueryStrKV {
			request.URI().QueryArgs().SetBytesV(kv.Val, request.URI().QueryArgs().Peek(kv.Key))
			request.URI().QueryArgs().Del(kv.Key)
		}
	}
	var modifyBody bool
	// Rename query body parameters
	if len(options.RenameBodyKV) != 0 {
		for _, kv := range options.RenameBodyKV {
			renameBody(ctx, request, kv)
		}
		modifyBody = true
	}
	return modifyBody
}

func renameBody(ctx context.Context, request *fasthttp.Request, kv *KV) {
	// Handle different content types
	if bytes.HasPrefix(request.Header.ContentType(), strPostArgsContentType) {
		request.PostArgs().SetBytesV(kv.Val, request.PostArgs().Peek(kv.Key))
		request.PostArgs().Del(kv.Key)
		return
	}
	if bytes.HasPrefix(request.Header.ContentType(), strJSONContentType) {
		newBody, _ := sjson.SetBytes(request.Body(), kv.Val, gjson.GetBytes(request.Body(), kv.Key).Value())
		newBody, _ = sjson.DeleteBytes(newBody, kv.Key)
		request.SetBody(newBody)
		return
	}
	if bytes.HasPrefix(request.Header.ContentType(), strMultipartFormData) {
		form, err := request.MultipartForm()
		if err != nil {
			log.ErrorContextf(ctx, "get multipart error: %s", err)
			return
		}
		form.Value[kv.Val] = form.Value[kv.Key]
		delete(form.Value, kv.Key)
		return
	}
	// Do nothing for other content types
	log.WarnContextf(ctx, "unsupported content-type: %s", request.Header.ContentType())
	return
}

// Delete operation
func deleteOption(ctx context.Context, request *fasthttp.Request, options *Options) bool {
	// Delete headers
	delHeaders(request, options)
	// Delete query parameters
	delQueryArgs(request, options)
	// Delete query body parameters
	return delBody(ctx, request, options)
}

// Delete body parameters
func delBody(ctx context.Context, request *fasthttp.Request, options *Options) bool {
	// Handle different content types
	if bytes.HasPrefix(request.Header.ContentType(), strPostArgsContentType) {
		return delQueryBody(ctx, request, options)
	}
	if bytes.HasPrefix(request.Header.ContentType(), strJSONContentType) {
		return delJSONBody(ctx, request, options)
	}
	if bytes.HasPrefix(request.Header.ContentType(), strMultipartFormData) {
		return delMultipartFormBody(ctx, request, options)
	}
	// Do nothing for other content types
	log.WarnContextf(ctx, "unsupported content-type: %s", request.Header.ContentType())
	return false
}

// Delete JSON body parameters
func delJSONBody(_ context.Context, request *fasthttp.Request, options *Options) bool {
	// Only keep high-priority operations
	if len(options.ReserveBody) != 0 {
		if options.ReserveBody[0] == DelAllKey {
			request.SetBodyString("")
			return true
		}
		reserveArgs := make(map[string]interface{}, len(options.ReserveBody))
		for _, a := range options.ReserveBody {
			reserveArgs[a] = gjson.GetBytes(request.Body(), a).Value()
		}
		var tmpBody []byte
		for k, v := range reserveArgs {
			tmpBody, _ = sjson.SetBytes(tmpBody, k, v)
		}
		request.SetBody(tmpBody)
		return true
	}
	// Specify parameters to remove
	if len(options.RemoveBody) != 0 {
		tmpBody := request.Body()
		for _, v := range options.RemoveBody {
			tmpBody, _ = sjson.DeleteBytes(tmpBody, v)
		}
		request.SetBody(tmpBody)
		return true
	}
	return false
}

// Delete MultipartForm body parameters
func delMultipartFormBody(ctx context.Context, request *fasthttp.Request, options *Options) bool {
	// Only keep, not supported for multiform type requests
	if len(options.ReserveBody) != 0 {
		return false
	}
	// Specify parameters to remove
	if len(options.RemoveBody) != 0 {
		form, err := request.MultipartForm()
		if err != nil {
			log.ErrorContextf(ctx, "get multipart form error: %s", err)
			return false
		}
		for _, v := range options.RemoveBody {
			delete(form.Value, v)
		}
		return true
	}
	return false
}

// Delete body query parameters
func delQueryBody(_ context.Context, request *fasthttp.Request, options *Options) bool {
	// Only keep high-priority operations
	if len(options.ReserveBody) != 0 {
		if options.ReserveBody[0] == DelAllKey {
			request.PostArgs().Reset()
			return true
		}
		reserveArgs := make(map[string][]byte, len(options.ReserveBody))
		for _, a := range options.ReserveBody {
			reserveArgs[a] = request.PostArgs().Peek(a)
		}
		request.PostArgs().Reset()
		for k, v := range reserveArgs {
			request.PostArgs().SetBytesV(k, v)
		}
		return true
	}
	// Specify parameters to remove
	if len(options.RemoveBody) != 0 {
		for _, v := range options.RemoveBody {
			request.PostArgs().Del(v)
		}
		return true
	}
	return false
}

// Delete headers
func delHeaders(request *fasthttp.Request, options *Options) {
	// Only keep high-priority operations
	if len(options.ReserveHeaders) != 0 {
		if options.ReserveHeaders[0] == DelAllKey {
			request.Header.Reset()
			return
		}
		reserveArgs := make(map[string][]byte, len(options.ReserveHeaders))
		for _, a := range options.ReserveHeaders {
			reserveArgs[a] = request.Header.Peek(a)
		}
		request.Header.Reset()
		for k, v := range reserveArgs {
			request.Header.SetBytesV(k, v)
		}
		return
	}
	if len(options.RemoveHeaders) != 0 {
		for _, v := range options.RemoveHeaders {
			request.Header.Del(v)
		}
	}
}

// Delete query parameters
func delQueryArgs(request *fasthttp.Request, options *Options) {
	// Only keep high-priority operations
	if len(options.ReserveQueryStr) != 0 {
		if options.ReserveQueryStr[0] == DelAllKey {
			request.URI().QueryArgs().Reset()
			return
		}
		reserveArgs := make(map[string][]byte, len(options.ReserveQueryStr))
		for _, a := range options.ReserveQueryStr {
			reserveArgs[a] = request.URI().QueryArgs().Peek(a)
		}
		request.URI().QueryArgs().Reset()
		for k, v := range reserveArgs {
			request.URI().QueryArgs().SetBytesV(k, v)
		}
		return
	}

	// Delete query parameters
	if len(options.RemoveQueryStr) != 0 {
		for _, v := range options.RemoveQueryStr {
			request.URI().QueryArgs().Del(v)
		}
	}
}
