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

// Package response response transformer plugin
package response

import (
	"context"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
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
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"
)

const (
	// PluginName response transformer
	PluginName = "response_transformer"

	// ErrInvalidJSONType invalid value type
	ErrInvalidJSONType = 10002
)

func init() {
	plugin.Register(PluginName, &Plugin{})
}

// Plugin response transformer plugin implementation
type Plugin struct {
}

// Type response transformer plugin type
func (p *Plugin) Type() string {
	return cplugin.DefaultType
}

// Setup response transformer instance initialization
func (p *Plugin) Setup(string, plugin.Decoder) error {
	filter.Register(PluginName, ServerFilter, nil)
	return nil
}

// KV key-value structure
type KV struct {
	Key string
	Val string
	// ConvertedVal for Val
	ConvertedVal interface{}
}

// valType JSON field type
type valType string

const (
	typeString  valType = "string"
	typeNumber  valType = "number"
	typeBool    valType = "bool"
	typeDefault valType = ""
)

// KeysConfig configuration
type KeysConfig struct {
	Keys          []string                        `yaml:"keys,omitempty" json:"keys,omitempty"`
	KVs           []*KV                           `yaml:"-"`
	StatusCodes   []int                           `yaml:"status_codes,omitempty" json:"status_codes,omitempty"`
	StatusCodeMap map[int]struct{}                `yaml:"-"`
	TRPCCodes     []trpcpb.TrpcRetCode            `yaml:"trpc_codes,omitempty" json:"trpc_codes,omitempty"`
	TRPCCodeMap   map[trpcpb.TrpcRetCode]struct{} `yaml:"-"`
}

// Options parameter options
type Options struct {
	// Remove operations
	// Remove response headers, in the format of header:status_code, e.g., traceid:401,404 means removing traceid when
	// the HTTP status code is 401	// If status_code is not specified, it means all statuses
	RemoveHeaders []*KeysConfig `yaml:"remove_headers,omitempty" json:"remove_headers,omitempty"`

	// Remove keys in the JSON body, in the format of key:status_code, e.g., suid:401 means removing suid when the HTTP
	// status code is 401
	// If status_code is not specified, it means all statuses
	// Key supports hierarchical configuration, e.g., common.suid
	RemoveJSON []*KeysConfig `yaml:"remove_json,omitempty" json:"remove_json,omitempty"`

	// Rename header configuration fields, in the format of old_header:new_header:status_code
	// Renamed headers will not be formatted
	RenameHeaders []*KeysConfig `yaml:"rename_headers,omitempty" json:"rename_headers,omitempty"`

	// Rename JSON body parameters, in the format of key:val:status_code, e.g., suid:xxx
	RenameJSON []*KeysConfig `yaml:"rename_json,omitempty" json:"rename_json,omitempty"`

	// Add headers, in the format of key:val
	AddHeaders []*KeysConfig `yaml:"add_headers,omitempty" json:"add_headers,omitempty"`

	// Add JSON body parameters, in the format of key:val:type, e.g., suid:xxx:string
	AddJSON []*KeysConfig `yaml:"add_json,omitempty" json:"add_json,omitempty"`

	// Replace operations
	ReplaceHeaders []*KeysConfig `yaml:"replace_headers,omitempty" json:"replace_headers,omitempty"`

	// key:val:type format, e.g., suid:xxx:string
	ReplaceJSON []*KeysConfig `yaml:"replace_json,omitempty" json:"replace_json,omitempty"`

	// Append operations
	AppendHeaders []*KeysConfig `yaml:"append_headers,omitempty" json:"append_headers,omitempty"`

	// key:val:type format, e.g., suid:xxx:string
	AppendJSON []*KeysConfig `yaml:"append_json,omitempty" json:"append_json,omitempty"`

	// Replace response body
	ReplaceBody []*KeysConfig `yaml:"replace_body,omitempty" json:"replace_body,omitempty"`

	// Allowed parameters
	AllowJSON []*KeysConfig `yaml:"allow_json,omitempty" json:"allow_json,omitempty"`
}

// CheckConfig validates the plugin configuration and returns the parsed configuration object with types. Used in the
// ServerFilter method for parsing.
func (p *Plugin) CheckConfig(name string, decoder plugin.Decoder) error {
	options := &Options{}
	if err := decoder.Decode(options); err != nil {
		return gerrs.Wrap(err, "decode response transformer config error")
	}
	log.Infof("plugin %s config: %s", name, convert.ToJSONStr(options))
	// Parse configuration
	if err := parseKV(options.RemoveHeaders, false); err != nil {
		return gerrs.Wrap(err, "remove header config error")
	}
	if err := parseKV(options.RemoveJSON, false); err != nil {
		return gerrs.Wrap(err, "remove JSON config error")
	}
	if err := parseKV(options.RenameHeaders, true); err != nil {
		return gerrs.Wrap(err, "rename header config error")
	}
	if err := parseKV(options.RenameJSON, true); err != nil {
		return gerrs.Wrap(err, "rename JSON config error")
	}
	if err := parseKV(options.AddHeaders, true); err != nil {
		return gerrs.Wrap(err, "add headers config error")
	}
	if err := parseKV(options.AddJSON, true); err != nil {
		return gerrs.Wrap(err, "add JSON config error")
	}
	if err := parseKV(options.ReplaceHeaders, true); err != nil {
		return gerrs.Wrap(err, "replace headers config error")
	}
	if err := parseKV(options.ReplaceJSON, true); err != nil {
		return gerrs.Wrap(err, "replace JSON config error")
	}
	if err := parseKV(options.AppendHeaders, true); err != nil {
		return gerrs.Wrap(err, "append headers config error")
	}
	if err := parseKV(options.AppendJSON, true); err != nil {
		return gerrs.Wrap(err, "append JSON config error")
	}
	if err := parseKV(options.AllowJSON, false); err != nil {
		return gerrs.Wrap(err, "allow JSON config error")
	}
	if err := parseKV(options.ReplaceBody, false); err != nil {
		return gerrs.Wrap(err, "replace body config error")
	}
	return nil
}

// Parse KV
func parseKV(list []*KeysConfig, isPair bool) error {
	for _, keys := range list {
		kvs, err := getKV(keys.Keys, isPair)
		if err != nil {
			return gerrs.Wrapf(err, "get KV error: %v", keys.Keys)
		}
		keys.KVs = kvs
		if len(keys.StatusCodes) > 0 {
			statusCodeMap := make(map[int]struct{})
			for _, code := range keys.StatusCodes {
				statusCodeMap[code] = struct{}{}
			}
			keys.StatusCodeMap = statusCodeMap
		}
		if len(keys.TRPCCodes) > 0 {
			tRPCCodeMap := make(map[trpcpb.TrpcRetCode]struct{})
			for _, code := range keys.TRPCCodes {
				tRPCCodeMap[code] = struct{}{}
			}
			keys.TRPCCodeMap = tRPCCodeMap
		}
	}
	return nil
}

// Get KV configuration
func getKV(list []string, isPair bool) ([]*KV, error) {
	var kvList []*KV
	for _, v := range list {
		if v == "" {
			continue
		}
		kv := &KV{}
		if !isPair {
			kv.Key = v
			kvList = append(kvList, kv)
			continue
		}
		arr := strings.Split(v, ":")
		if len(arr) < 2 {
			return nil, errs.New(gerrs.ErrWrongConfig, "invalid KV configuration")
		}

		if arr[0] == "" {
			continue
		}
		kv.Key = arr[0]
		kv.Val = arr[1]
		if len(arr) == 2 {
			kv.ConvertedVal = kv.Val
			kvList = append(kvList, kv)
			continue
		}
		// Convert based on the configured type
		convertedVal, err := convertJSONVal(kv.Val, valType(arr[2]))
		if err != nil {
			return nil, gerrs.Wrap(err, "convert JSON value error")
		}
		kv.ConvertedVal = convertedVal
		kvList = append(kvList, kv)
	}
	return kvList, nil
}

// Convert the type of JSON value
func convertJSONVal(val string, valType valType) (interface{}, error) {
	switch valType {
	case typeString:
		return fmt.Sprint(val), nil
	case typeNumber:
		floatValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, gerrs.Wrapf(err, "to float err,val:%s", val)
		}
		return floatValue, nil
	case typeBool:
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return nil, gerrs.Wrapf(err, "parse bool error: %s", val)
		}
		return boolVal, nil
	case typeDefault:
		return val, nil
	default:
		return nil, errs.Newf(ErrInvalidJSONType, "invalid type: %s", valType)
	}
}

// ServerFilter sets server-side CORS verification
func ServerFilter(ctx context.Context, req interface{}, handler filter.ServerHandleFunc) (interface{}, error) {
	rsp, err := handler(ctx, req)
	if terr := transform(ctx, errs.Code(err)); terr != nil {
		// Transformation failed, handle degradation
		log.ErrorContextf(ctx, "transform response error: %s", terr)
	}
	return rsp, err
}

// Request transformation
func transform(ctx context.Context, tRPCCode trpcpb.TrpcRetCode) error {
	defer func() {
		if r := recover(); r != nil {
			log.ErrorContextf(ctx, "response transformer handle panic: %s", string(debug.Stack()))
		}
	}()
	// Parse plugin configuration
	pluginConfig := gwmsg.GwMessage(ctx).PluginConfig(PluginName)
	if pluginConfig == nil {
		return errs.New(gerrs.ErrPluginConfigNotFound, "no response transformer plugin config found")
	}
	options, ok := pluginConfig.(*Options)
	if !ok {
		return errs.New(gerrs.ErrPluginConfigNotFound, "invalid response transformer plugin config type")
	}
	// Modify the response content
	if err := modifyResponse(ctx, options, tRPCCode); err != nil {
		return gerrs.Wrap(err, "modify response error")
	}
	return nil
}

// Modify the response content
func modifyResponse(ctx context.Context, options *Options, tRPCCode trpcpb.TrpcRetCode) error {
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		// Not an HTTP request, do not process
		return nil
	}
	if err := deleteOption(&fctx.Response, options, tRPCCode); err != nil {
		return gerrs.Wrap(err, "delete option error")
	}
	if err := renameOption(&fctx.Response, options, tRPCCode); err != nil {
		return gerrs.Wrap(err, "rename option error")
	}
	if err := replaceOption(&fctx.Response, options, tRPCCode); err != nil {
		return gerrs.Wrap(err, "replace option error")
	}
	if err := addOption(&fctx.Response, options, tRPCCode); err != nil {
		return gerrs.Wrap(err, "add option error")
	}
	if err := appendOption(&fctx.Response, options, tRPCCode); err != nil {
		return gerrs.Wrap(err, "append option error")
	}
	if err := replaceBodyOption(&fctx.Response, options, tRPCCode); err != nil {
		return gerrs.Wrap(err, "replace body error")
	}
	if err := allowJSONOption(&fctx.Response, options, tRPCCode); err != nil {
		return gerrs.Wrap(err, "allow JSON error")
	}
	return nil
}

// Delete operation
func deleteOption(response *fasthttp.Response, options *Options, tRPCCode trpcpb.TrpcRetCode) error {
	// Delete headers
	if err := iterateKV(options.RemoveHeaders, response, tRPCCode, removeHeaders); err != nil {
		return gerrs.Wrap(err, "remove header error")
	}

	// Delete JSON keys
	if err := iterateKV(options.RemoveJSON, response, tRPCCode, removeJSON); err != nil {
		return gerrs.Wrap(err, "remove JSON error")
	}
	return nil
}

// Remove response headers
func removeHeaders(response *fasthttp.Response, kv *KV) error {
	response.Header.Del(kv.Key)
	return nil
}

// Remove response body fields
func removeJSON(response *fasthttp.Response, kv *KV) error {
	nb, err := sjson.DeleteBytes(response.Body(), kv.Key)
	if err != nil {
		return gerrs.Wrap(err, "delete JSON body key error")
	}
	response.SetBody(nb)
	// Reset Content-Length
	response.Header.SetContentLength(len(nb))
	return nil
}

// Rename operation
func renameOption(response *fasthttp.Response, options *Options, tRPCCode trpcpb.TrpcRetCode) error {
	// Rename headers
	if err := iterateKV(options.RenameHeaders, response, tRPCCode, renameHeaders); err != nil {
		return gerrs.Wrap(err, "rename header error")
	}

	// Rename JSON keys
	if err := iterateKV(options.RenameJSON, response, tRPCCode, renameJSON); err != nil {
		return gerrs.Wrap(err, "rename JSON error")
	}
	return nil
}

// Rename response headers
func renameHeaders(response *fasthttp.Response, kv *KV) error {
	val := response.Header.Peek(kv.Key)
	if len(val) == 0 {
		return nil
	}
	response.Header.DisableNormalizing()
	response.Header.SetBytesV(kv.Val, val)
	response.Header.EnableNormalizing()
	response.Header.Del(kv.Key)
	return nil
}

// Rename response body fields
func renameJSON(response *fasthttp.Response, kv *KV) error {
	currVal := gjson.GetBytes(response.Body(), kv.Key)
	if !currVal.Exists() {
		return nil
	}
	nb, err := sjson.SetBytes(response.Body(), kv.Val, currVal.Value())
	if err != nil {
		return gerrs.Wrap(err, "set JSON error")
	}
	nb, err = sjson.DeleteBytes(nb, kv.Key)
	if err != nil {
		return gerrs.Wrap(err, "delete JSON error")
	}
	response.SetBody(nb)
	// Reset Content-Length
	response.Header.SetContentLength(len(nb))
	return nil
}

// Replace operation
func replaceOption(response *fasthttp.Response, options *Options, tRPCCode trpcpb.TrpcRetCode) error {
	// Replace headers
	if err := iterateKV(options.ReplaceHeaders, response, tRPCCode, replaceHeader); err != nil {
		return gerrs.Wrap(err, "replace header error")
	}
	// Replace JSON
	if err := iterateKV(options.ReplaceJSON, response, tRPCCode, replaceJSON); err != nil {
		return gerrs.Wrap(err, "replace JSON error")
	}
	return nil
}

// Replace response headers
func replaceHeader(response *fasthttp.Response, kv *KV) error {
	val := response.Header.Peek(kv.Key)
	if len(val) == 0 {
		return nil
	}
	response.Header.Set(kv.Key, kv.Val)
	return nil
}

// Replace response body fields
func replaceJSON(response *fasthttp.Response, kv *KV) error {
	if !gjson.GetBytes(response.Body(), kv.Key).Exists() {
		return nil
	}
	nb, err := sjson.SetBytes(response.Body(), kv.Key, kv.ConvertedVal)
	if err != nil {
		return gerrs.Wrap(err, "replace JSON error")
	}
	response.SetBody(nb)
	// Reset Content-Length
	response.Header.SetContentLength(len(nb))
	return nil
}

// Add operation
func addOption(response *fasthttp.Response, options *Options, tRPCCode trpcpb.TrpcRetCode) error {
	// Add headers
	if err := iterateKV(options.AddHeaders, response, tRPCCode, addHeaders); err != nil {
		return gerrs.Wrap(err, "add header error")
	}

	// Add JSON
	if err := iterateKV(options.AddJSON, response, tRPCCode, addJSON); err != nil {
		return gerrs.Wrap(err, "add JSON error")
	}
	return nil
}

// Add response headers
func addHeaders(response *fasthttp.Response, kv *KV) error {
	response.Header.Set(kv.Key, kv.Val)
	return nil
}

// Add response body fields
func addJSON(response *fasthttp.Response, kv *KV) error {
	nb, err := sjson.SetBytes(response.Body(), kv.Key, kv.ConvertedVal)
	if err != nil {
		return gerrs.Wrap(err, "add JSON error")
	}
	response.SetBody(nb)
	// Reset Content-Length
	response.Header.SetContentLength(len(nb))
	return nil
}

// Append operation
func appendOption(response *fasthttp.Response, options *Options, tRPCCode trpcpb.TrpcRetCode) error {
	// Append headers
	if err := iterateKV(options.AppendHeaders, response, tRPCCode, appendHeaders); err != nil {
		return gerrs.Wrap(err, "append header error")
	}

	// Append JSON
	if err := iterateKV(options.AppendJSON, response, tRPCCode, appendJSON); err != nil {
		return gerrs.Wrap(err, "append JSON error")
	}
	return nil
}

// Append response header fields
func appendHeaders(response *fasthttp.Response, kv *KV) error {
	response.Header.Add(kv.Key, kv.Val)
	return nil
}

// Append response body fields
func appendJSON(response *fasthttp.Response, kv *KV) error {
	r := gjson.GetBytes(response.Body(), kv.Key)
	if !r.Exists() {
		// If it doesn't exist, add it
		nb, err := sjson.SetBytes(response.Body(), kv.Key, kv.ConvertedVal)
		if err != nil {
			return gerrs.Wrap(err, "add JSON error")
		}
		response.SetBody(nb)
		// Reset Content-Length
		response.Header.SetContentLength(len(nb))
		return nil
	}
	// If it exists, append to it
	arr := []interface{}{r.Value(), kv.ConvertedVal}
	nb, err := sjson.SetBytes(response.Body(), kv.Key, arr)
	if err != nil {
		return gerrs.Wrap(err, "append JSON error")
	}
	response.SetBody(nb)
	// Reset Content-Length
	response.Header.SetContentLength(len(nb))
	return nil
}

// Replace body operation
func replaceBodyOption(response *fasthttp.Response, options *Options, tRPCCode trpcpb.TrpcRetCode) error {
	// Replace response body
	err := iterateKV(options.ReplaceBody, response, tRPCCode, replaceBody)
	if err != nil {
		return gerrs.Wrap(err, "replace body error")
	}
	return nil
}

// Replace response body
func replaceBody(response *fasthttp.Response, kv *KV) error {
	response.SetBodyString(kv.Key)
	response.Header.SetContentLength(len(response.Body()))
	return nil
}

// Limit response body fields
func allowJSONOption(response *fasthttp.Response, options *Options, tRPCCode trpcpb.TrpcRetCode) error {
	// Append headers
	for _, v := range options.AllowJSON {
		// Validate status code
		if !validCode(v.StatusCodeMap, v.TRPCCodeMap, response.StatusCode(), tRPCCode) {
			continue
		}
		if len(v.KVs) == 0 {
			continue
		}
		var keyList []string
		for _, kV := range v.KVs {
			keyList = append(keyList, kV.Key)
		}
		var nb []byte
		var err error
		for idx, val := range gjson.GetManyBytes(response.Body(), keyList...) {
			nb, err = sjson.SetBytes(nb, keyList[idx], val.Value())
			if err != nil {
				return gerrs.Wrap(err, "set JSON value error")
			}
		}
		response.SetBody(nb)
		response.Header.SetContentLength(len(nb))
	}
	return nil
}

// Handler function for various operations
type handlerFun func(response *fasthttp.Response, kv *KV) error

// Iterate over all keys
func iterateKV(keys []*KeysConfig, response *fasthttp.Response, tRPCCode trpcpb.TrpcRetCode, handle handlerFun) error {
	for _, v := range keys {
		// Validate status code
		if !validCode(v.StatusCodeMap, v.TRPCCodeMap, response.StatusCode(), tRPCCode) {
			continue
		}
		for _, kV := range v.KVs {
			err := handle(response, kV)
			if err != nil {
				return gerrs.Wrap(err, "handle error")
			}
		}
	}
	return nil
}

// Validate if the status code meets the requirements, return true if both are empty
func validCode(statusCodeMap map[int]struct{}, tRPCCodeMap map[trpcpb.TrpcRetCode]struct{}, statusCode int,
	tRPCCode trpcpb.TrpcRetCode) bool {
	var valid bool
	if len(statusCodeMap) != 0 {
		valid = true
		if _, ok := statusCodeMap[statusCode]; ok {
			return true
		}
	}

	if len(tRPCCodeMap) != 0 {
		valid = true
		if _, ok := tRPCCodeMap[tRPCCode]; ok {
			return true
		}
	}
	return !valid
}
