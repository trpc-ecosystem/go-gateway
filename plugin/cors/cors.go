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

// Package cors provides cross-origin resource sharing for HTTP requests.
package cors

import (
	"context"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"

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
	pluginName = "cors"
)

var (
	defaultAllowMethods = []string{
		fasthttp.MethodGet,
		fasthttp.MethodHead,
		fasthttp.MethodPut,
		fasthttp.MethodPost,
		fasthttp.MethodDelete,
		fasthttp.MethodPatch,
	}
	defaultExposeHeaders = []string{
		"trpc-version",
		"trpc-call-type",
		"trpc-request-id",
		"trpc-ret",
		"trpc-func-ret",
		"trpc-message-type",
		"trpc-error-msg",
		"trpc-trans-info",
	}
)

const (
	// Reference for response headers: https://developer.mozilla.org/zh-CN/docs/Web/HTTP/CORS
	// Common response headers for cross-origin requests
	accessControlAllowOrigin      = "Access-Control-Allow-Origin"
	timingAllowOrigin             = "Timing-Allow-Origin"
	accessControlAllowCredentials = "Access-Control-Allow-Credentials"

	// Expose header response header
	accessControlExposeHeaders = "Access-Control-Expose-Headers"

	// Preflight response headers
	accessControlRequestMethod  = "Access-Control-Request-Method"
	accessControlRequestHeaders = "Access-Control-Request-Headers"
	accessControlAllowMethods   = "Access-Control-Allow-Methods"
	accessControlAllowHeaders   = "Access-Control-Allow-Headers"
	accessControlMaxAge         = "Access-Control-Max-Age"
	vary                        = "Vary"
	origin                      = "Origin"

	// ErrCORS is the error code for cross-origin requests. The error codes in the gateway plugin are custom and use 5
	// digits.
	ErrCORS = 10000
)

func init() {
	plugin.Register(pluginName, &Plugin{})
}

// Plugin implements the cors plugin for trpc.
type Plugin struct {
}

// Type returns the type of the cors plugin for trpc.
func (p *Plugin) Type() string {
	return cplugin.DefaultType
}

// Setup initializes the cors plugin instance.
func (p *Plugin) Setup(string, plugin.Decoder) error {
	filter.Register(pluginName, ServerFilter, nil)
	if err := gerrs.Register(ErrCORS, fasthttp.StatusForbidden); err != nil {
		return gerrs.Wrap(err, "register cors code err")
	}
	return nil
}

// Options represents the parameter options for the cors plugin.
type Options struct {
	// AllowOrigins is a list of allowed cross-origin request sources. If empty, it allows all cross-origin requests,
	// i.e., wildcard *.
	AllowOrigins []string `yaml:"allow_origins"`

	// AllowMethods is a list of allowed cross-origin request methods. If not specified, the default value is
	// []string{"GET", "HEAD", "PUT", "POST", "DELETE", "PATCH"}.
	AllowMethods []string `yaml:"allow_methods"`

	// AllowHeaders is a list of headers that can be used in the actual request returned by the preflight request.
	// If not specified, it allows all preflight request headers.
	AllowHeaders []string `yaml:"allow_headers"`

	// AllowCredentials specifies whether to allow credentials to be carried, such as cookies.
	AllowCredentials bool `yaml:"allow_credentials"`

	// ExposeHeaders is a list of custom headers that can be obtained in cross-origin requests, such as trpc-ret.
	ExposeHeaders []string `yaml:"expose_headers"`

	// MaxAge is the cache time for preflight requests, in seconds.
	MaxAge int `yaml:"max_age"`
}

// CheckConfig validates the plugin configuration and returns the parsed configuration object. Used in the ServerFilter
// method for parsing.
func (p *Plugin) CheckConfig(name string, decoder plugin.Decoder) error {
	options := &Options{}
	if err := decoder.Decode(options); err != nil {
		return gerrs.Wrap(err, "decode cors config err")
	}
	log.Infof("plugin %s config:%s", name, convert.ToJSONStr(options))
	// Set default request methods
	if len(options.AllowMethods) == 0 {
		options.AllowMethods = defaultAllowMethods
	}

	// Set exposed headers and remove duplicates
	options.ExposeHeaders = append(options.ExposeHeaders, defaultExposeHeaders...)
	var exposeHeaders []string
	for h := range convert.StrSlice2Map(options.ExposeHeaders) {
		exposeHeaders = append(exposeHeaders, h)
	}
	options.ExposeHeaders = exposeHeaders

	// Validate plugin parameters
	// Cannot set both Credentials and *
	if options.AllowCredentials && len(options.AllowOrigins) == 0 {
		return errs.New(gerrs.ErrWrongConfig,
			"can't set Origin to * and AllowCredentials to true at the same time")
	}

	return nil
}

// ServerFilter sets up server-side CORS validation.
// Reference for CORS headers: https://developer.mozilla.org/zh-CN/docs/Web/HTTP/CORS
func ServerFilter(ctx context.Context, req interface{}, handler filter.ServerHandleFunc) (interface{}, error) {
	defer func() {
		if r := recover(); r != nil {
			log.ErrorContextf(ctx, "cors handle panic:%s", string(debug.Stack()))
		}
	}()

	// Parse plugin configuration
	pluginConfig := gwmsg.GwMessage(ctx).PluginConfig(pluginName)
	if pluginConfig == nil {
		return nil, errs.New(gerrs.ErrPluginConfigNotFound, "get no cors plugin config")
	}
	corsConf, ok := pluginConfig.(*Options)
	if !ok {
		return nil, errs.New(gerrs.ErrPluginConfigNotFound, "invalid cors plugin config type")
	}

	// If it's not an HTTP request, no CORS validation is performed by default
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return handler(ctx, req)
	}

	// Cross-origin request origin
	origin := string(fctx.Request.Header.Peek(origin))
	if len(origin) == 0 { // Not a cross-origin request, no need to add CORS headers
		return handler(ctx, req)
	}

	parsedOriginObj, err := url.Parse(origin)
	if err != nil { // Error parsing Origin, no need to add CORS headers
		return handler(ctx, req)
	}

	// Check if it's in the supported CORS configuration
	if len(corsConf.AllowOrigins) != 0 && !isAllowCORS(corsConf.AllowOrigins, parsedOriginObj.Host) {
		return nil, errs.New(ErrCORS, "cors not allow")
	}

	// Determine the allowed resource address
	allowOrigin := origin
	if len(corsConf.AllowOrigins) == 0 {
		// Allow all cross-origin requests if not configured
		allowOrigin = "*"
	}

	// Handle preflight request
	if string(fctx.Method()) == fasthttp.MethodOptions {
		// Set common response headers
		addCommonHeader(fctx, allowOrigin, corsConf.AllowCredentials)
		// Set preflight response headers
		addPreflightHeader(fctx, corsConf)
		// Return directly for preflight requests
		return nil, nil
	}

	rsp, err := handler(ctx, req)
	// Set common response headers
	addCommonHeader(fctx, allowOrigin, corsConf.AllowCredentials)
	// Set actual request response headers
	addActualReqHeader(fctx, corsConf.ExposeHeaders)
	// Proceed with subsequent flow after successful authentication
	return rsp, err
}

// addActualReqHeader sets the response headers for the actual request.
func addActualReqHeader(fctx *fasthttp.RequestCtx, exposeHeaders []string) {
	if len(exposeHeaders) > 0 {
		fctx.Response.Header.Set(accessControlExposeHeaders, strings.Join(exposeHeaders, ", "))
	}
}

// addCommonHeader sets the common CORS response headers for both preflight and actual requests.
func addCommonHeader(fctx *fasthttp.RequestCtx, allowOrigin string, allowCredentials bool) {
	// Specify the allowed request origin. For security reasons, wildcard * is not allowed.
	fctx.Response.Header.Set(accessControlAllowOrigin, allowOrigin)
	// If the server specifies a specific single origin (as part of the allowed list, which may change dynamically
	// based on the request origin),
	// rather than the wildcard "", the value of the Vary header in the response must include Origin.
	// This tells the client that the server returns different content for different origins.
	fctx.Response.Header.Set(vary, origin)
	// The Timing-Allow-Origin response header is used to specify a specific site to allow access to the relevant
	// information provided by the Resource Timing API.
	// Otherwise, this information will be reported as zero due to cross-origin restrictions.
	fctx.Response.Header.Set(timingAllowOrigin, allowOrigin)
	if allowCredentials {
		// Set this response header for requests that carry credentials (such as cookies);
		// mutually exclusive with Access-Control-Allow-Origin: *
		fctx.Response.Header.Set(accessControlAllowCredentials, "true")
	}
}

// addPreflightHeader sets the response headers for preflight requests.
func addPreflightHeader(fctx *fasthttp.RequestCtx, o *Options) {
	// Set: Vary
	fctx.Response.Header.Set(vary, accessControlRequestMethod)
	fctx.Response.Header.Set(vary, accessControlRequestHeaders)

	// Set: Access-Control-Allow-Methods
	fctx.Response.Header.Set(accessControlAllowMethods, strings.Join(o.AllowMethods, ", "))

	// Set: Access-Control-Allow-Headers
	allowHeaders := ""
	if len(o.AllowHeaders) > 0 {
		allowHeaders = strings.Join(o.AllowHeaders, ", ")
	} else {
		// When AllowHeaders is nil, allow all headers in the request by default
		allowHeaders = string(fctx.Request.Header.Peek(accessControlRequestHeaders))
	}
	if allowHeaders != "" {
		fctx.Response.Header.Set(accessControlAllowHeaders, allowHeaders)
	}

	// Set: Access-Control-Max-Age
	// If MaxAge is set, it means the preflight request can be cached for the specified MaxAge time.
	if o.MaxAge > 0 {
		fctx.Response.Header.Set(accessControlMaxAge, strconv.Itoa(o.MaxAge))
	}
}

// isAllowCORS checks if the cross-origin request is allowed.
func isAllowCORS(configDomains []string, originHost string) bool {
	for _, domain := range configDomains {
		domainWithDot := "." + domain
		if strings.HasSuffix(originHost, domainWithDot) { // Match the parsed Origin Host with the suffix
			return true
		}
		if originHost == domain {
			return true
		}
	}
	return false
}
