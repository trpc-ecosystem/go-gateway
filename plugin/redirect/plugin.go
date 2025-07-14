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

// Package redirect implements redirection functionality.
package redirect

import (
	"context"
	"fmt"
	"regexp"
	"runtime/debug"
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
	pluginName = "redirect"
)

func init() {
	plugin.Register(pluginName, &Plugin{})
}

// Plugin is the plugin definition.
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

// Options is the plugin configuration.
type Options struct {
	// When set to true and the request is HTTP, it will be redirected to HTTPS with the same URI and 301 status code.
	HTTPToHTTPS bool `yaml:"http_to_https"`
	// The URI to redirect to.
	URI string `yaml:"uri"`
	// Match the URL from the client with regular expressions and redirect. After a successful match, replace the
	// client's request URI with the redirect URI template.
	// For example: ["^/iresty/(.)/(.)/(.*)","/$1-$2-$3"]
	// The first element represents the regular expression to match the URI from the client request,
	// and the second element represents the URI template to send the redirect to the client after a successful match.
	RegexURI         []string       `yaml:"regex_uri"`
	compiledRegexURI *regexp.Regexp `yaml:"-"`
	// HTTP response code.
	RetCode int `yaml:"ret_code"`
	// When set to true, append the query string from the original request to the Location Header.
	// If the uri or regex_uri already contains a query string, the query string from the request will be appended
	// with an "&".
	AppendQueryString bool `yaml:"append_query_string"`
}

// CheckConfig validates the plugin configuration and returns the parsed configuration object. Used in the
// ServerFilter method for parsing.
func (p *Plugin) CheckConfig(_ string, decoder plugin.Decoder) error {
	options := &Options{}
	if err := decoder.Decode(options); err != nil {
		return gerrs.Wrap(err, "decode dev env config err")
	}
	uriConfCount := 0
	if options.HTTPToHTTPS {
		uriConfCount++
	}

	if options.URI != "" {
		uriConfCount++
	}
	if len(options.RegexURI) != 0 {
		uriConfCount++
	}
	if uriConfCount > 1 {
		return errs.New(gerrs.ErrInvalidPluginConfig, "duplicated uri config")
	}

	if len(options.RegexURI) != 0 && len(options.RegexURI) != 2 {
		return errs.New(gerrs.ErrInvalidPluginConfig, "invalid regexp uri config")
	}
	if len(options.RegexURI) == 2 {
		if options.RegexURI[0] == "" || options.RegexURI[1] == "" {
			return errs.New(gerrs.ErrInvalidPluginConfig, "regexp uri can not be empty")
		}
		// Compile the regular expression
		r, err := regexp.Compile(options.RegexURI[0])
		if err != nil {
			return errs.Wrap(err, gerrs.ErrInvalidPluginConfig, "compile regexp uri err")
		}
		options.compiledRegexURI = r
	}
	if options.RetCode == 0 {
		options.RetCode = fasthttp.StatusFound
	}
	return nil
}

// ServerFilter is the server interceptor.
func ServerFilter(ctx context.Context, req interface{}, handle filter.ServerHandleFunc) (interface{}, error) {
	doRedirect, err := redirect(ctx)
	if err != nil {
		// Ignore redirect error
		log.ErrorContextf(ctx, "redirect err:%s", err)
	}
	if doRedirect {
		return nil, nil
	}
	return handle(ctx, req)
}

// Override forwarding environment
func redirect(ctx context.Context) (bool, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error(ctx, "redirect panic:%s,stack:%s", r, debug.Stack())
		}
	}()

	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return false, nil
	}
	// Parse plugin configuration
	pluginConfig := gwmsg.GwMessage(ctx).PluginConfig(pluginName)
	if pluginConfig == nil {
		return false, errs.New(gerrs.ErrWrongConfig, "get no redirect config")
	}
	options, ok := pluginConfig.(*Options)
	if !ok {
		log.ErrorContextf(ctx, "invalid redirect config")
		return false, errs.New(gerrs.ErrWrongConfig, "invalid redirect config")
	}

	log.DebugContextf(ctx, "redirect config:%s", convert.ToJSONStr(options))
	// Check http2https, uri, and regexp_uri one by one. Return true if any of them satisfies the condition.
	targetURI, err := getTargetURI(fctx, options)
	if err != nil {
		return false, gerrs.Wrap(err, "get target uri err")
	}
	log.DebugContextf(ctx, "target uri:%s", targetURI)
	// If no target URI is obtained, do not redirect
	if targetURI == "" {
		return false, nil
	}
	strLocation := []byte("Location")
	fctx.Response.Header.SetCanonical(strLocation, []byte(targetURI))
	fctx.Response.SetStatusCode(options.RetCode)
	return true, nil
}

func getTargetURI(fctx *fasthttp.RequestCtx, options *Options) (string, error) {
	targetURI, err := rebuildURI(fctx, options)
	if err != nil {
		return "", gerrs.Wrap(err, "get target path err")
	}
	if targetURI == "" {
		return "", nil
	}
	if !options.AppendQueryString {
		return targetURI, nil
	}
	if len(fctx.Request.URI().QueryString()) == 0 {
		return targetURI, nil
	}
	if strings.Contains(targetURI, "?") {
		return fmt.Sprintf("%s&%s", targetURI, fctx.Request.URI().QueryString()), nil
	}
	return fmt.Sprintf("%s?%s", targetURI, fctx.Request.URI().QueryString()), nil
}

func rebuildURI(fctx *fasthttp.RequestCtx, options *Options) (string, error) {
	if options.HTTPToHTTPS {
		return fmt.Sprintf("https://%s%s", fctx.Host(), fctx.Path()), nil
	}
	if options.URI != "" {
		// The URI configured here may also contain query parameters
		return options.URI, nil
	}
	if len(options.RegexURI) == 2 {
		// Match and then replace
		if options.compiledRegexURI.MatchString(string(fctx.Path())) {
			if strings.Contains(options.RegexURI[1], "$") {
				return options.compiledRegexURI.ReplaceAllString(string(fctx.Path()), options.RegexURI[1]), nil
			}
			return options.RegexURI[1], nil
		}
		return "", nil
	}
	return "", errs.New(gerrs.ErrInvalidPluginConfig, "invalid redirect uri")
}
