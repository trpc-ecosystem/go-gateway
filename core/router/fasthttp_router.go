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

package router

import (
	"context"
	"fmt"
	"math/rand"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"

	radix "github.com/armon/go-radix"
	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/convert"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	cplugin "trpc.group/trpc-go/trpc-gateway/common/plugin"
	"trpc.group/trpc-go/trpc-gateway/core/config"
	"trpc.group/trpc-go/trpc-gateway/core/entity"
	"trpc.group/trpc-go/trpc-gateway/core/rule"
	"trpc.group/trpc-go/trpc-gateway/core/service/protocol"
	"trpc.group/trpc-go/trpc-gateway/internal/util"
	gwplugin "trpc.group/trpc-go/trpc-gateway/plugin"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
)

// DefaultFastHTTPRouter is the default FastHTTP router
var DefaultFastHTTPRouter = NewFastHTTPRouter()

const confModule = "router"

func init() {
	rand.Seed(time.Now().UnixNano())

	RegisterRouter(protocolFastHTTP, DefaultFastHTTPRouter)
	config.RegisterRouteLoader(protocolFastHTTP, DefaultFastHTTPRouter.LoadRouterConf)
}

// FastHTTPRouter is the FastHTTP router
type FastHTTPRouter struct {
	opts         *Options // proxy configuration
	sync.RWMutex          // read-write lock for updating configuration
}

// NewFastHTTPRouter creates a new FastHTTP router
func NewFastHTTPRouter() *FastHTTPRouter {
	return &FastHTTPRouter{opts: &Options{}}
}

// getOpts gets the proxy configuration.
// Use this method to access the configuration to avoid race conditions when directly reading opts
// during configuration updates.
func (r *FastHTTPRouter) getOpts() *Options {
	r.RLock()
	defer r.RUnlock()
	return r.opts
}

// setOpts sets the proxy configuration
func (r *FastHTTPRouter) setOpts(opts *Options) {
	r.Lock()
	defer r.Unlock()
	r.opts = opts
}

// LoadRouterConf loads the router configuration
func (r *FastHTTPRouter) LoadRouterConf(provider string) (err error) {
	provider = provider + "_" + confModule
	loader := config.GetConfLoader(provider)
	if loader == nil {
		return fmt.Errorf("provider %s does not exist", provider)
	}
	err = loader.LoadConf(context.Background(), protocolFastHTTP)
	if err != nil {
		return gerrs.Wrap(err, "load_fast_http_router_err")
	}
	return nil
}

// InitRouterConfig loads the initial router configuration
// 1. Initialize the trie tree, where the node values are target arrays
// 2. Initialize the regular expression matching, which is a list
// 3. Upstream service configuration
// 4. Plugin configuration
func (r *FastHTTPRouter) InitRouterConfig(ctx context.Context, rf *entity.ProxyConfig) error {
	options, err := r.CheckAndInit(ctx, rf)
	if err != nil {
		return gerrs.Wrap(err, "check and init err")
	}

	// Override the original options during router initialization
	r.setOpts(options)
	return nil
}

// CheckAndInit validates and initializes the configuration
func (r *FastHTTPRouter) CheckAndInit(ctx context.Context, rf *entity.ProxyConfig) (*Options, error) {
	defer func() {
		if r := recover(); r != nil {
			log.ErrorContextf(ctx, "check and init router config panic: %s, stack: %s", r, string(debug.Stack()))
		}
	}()
	// Proxy configuration
	options := &Options{
		RadixTree:     radix.New(),
		RegRouterList: []*RegRouter{},
		Clients:       map[string]*entity.BackendConfig{},
	}
	// Load upstream service configuration
	opts := r.getTargetServiceOpts(ctx, rf)
	for _, o := range opts {
		o(options)
	}

	// Load router configuration
	routerOpts, err := r.getRouterOpts(ctx, rf, options)
	if err != nil {
		return nil, gerrs.Wrap(err, "get router opts err")
	}
	for _, o := range routerOpts {
		o(options)
	}
	return options, nil
}

// getTargetServiceOpts gets the backend service configuration
func (r *FastHTTPRouter) getTargetServiceOpts(ctx context.Context, rf *entity.ProxyConfig) []Option {
	var opts []Option
	if len(rf.Client) == 0 {
		log.WarnContextf(ctx, "empty client config!!")
		return opts
	}
	for _, cli := range rf.Client {
		opts = append(opts, WithRouterClient(cli))
	}
	return opts
}

// getRouterOpts retrieves the router configuration
func (r *FastHTTPRouter) getRouterOpts(ctx context.Context, rf *entity.ProxyConfig, options *Options) ([]Option, error) {
	var opts []Option
	// If the router is empty, throw an error: for security reasons, to prevent the entire router from being
	// set to empty due to incorrect reading of the configuration.
	if len(rf.Router) == 0 {
		log.ErrorContextf(ctx, "Empty router configuration! Requires at least one router")
		return opts, errs.New(gerrs.ErrWrongConfig, "empty router configuration")
	}
	for _, routerItem := range rf.Router {
		// Initialize the upstream service configuration
		if err := r.initTargetService(routerItem.TargetService, options.Clients, routerItem.Plugins, rf.Plugins); err != nil {
			return nil, gerrs.Wrapf(err, "init target service error")
		}
		// Method cannot be empty or "/"
		if routerItem.Method == "" || routerItem.Method == "/" {
			return nil, errs.Newf(gerrs.ErrWrongConfig, "invalid method configuration: %s", convert.ToJSONStr(routerItem))
		}
		// Validate and parse the condition expression
		if routerItem.Rule != nil && routerItem.Rule.Expression != "" {
			if err := rule.FormatRule(routerItem.Rule); err != nil {
				return nil, gerrs.Wrap(err, "format rule error")
			}
		}
		// Convert to a map
		routerItem.HostMap = convert.StrSlice2Map(routerItem.Host)

		// Check if it is a regular expression router
		if routerItem.IsRegexp {
			opts = append(opts, WithRegRouter(routerItem))
			continue
		}
		// If it is not a regular expression router, it is either an exact match or a prefix match,
		// and it is added to the trie tree
		opts = append(opts, WithRadixTreeRouter(routerItem))
	}
	return opts, nil
}

// initTargetService initializes the upstream service configuration
func (r *FastHTTPRouter) initTargetService(targetServiceList []*entity.TargetService,
	clientMap map[string]*entity.BackendConfig, routerPlugins, globalPlugins []*entity.Plugin) error {
	if len(targetServiceList) == 0 {
		return errs.New(gerrs.ErrWrongConfig, "empty target service")
	}
	var totalWeight int
	for _, s := range targetServiceList {
		// Accumulate weights
		totalWeight += s.Weight
		service, err := r.checkService(clientMap, s.Service)
		if err != nil {
			return gerrs.Wrap(err, "check service error")
		}
		s.BackendConfig = &service.BackendConfig
		// Merge gateway plugins at global, service, and router levels
		s.Plugins = r.mergePlugins(routerPlugins, service.Plugins, globalPlugins)
		// Iterate through all plugins and parse their configurations
		var pluginsNameList []string
		for _, pluginConfig := range s.Plugins {
			parsedConfig, perr := r.parsePluginConfig(pluginConfig)
			if perr != nil {
				return gerrs.Wrapf(perr, "parse plugin config error")
			}
			pluginConfig.Props = parsedConfig
			pluginsNameList = append(pluginsNameList, pluginConfig.Name)
		}
		// Assemble all filters, with trpc filters first and deduplicated
		for _, name := range util.Deduplicate(trpc.GlobalConfig().Server.Filter, pluginsNameList) {
			log.Debugf("plugin_name:%s", name)
			filterFunc := filter.GetServer(name)
			if filterFunc == nil {
				return gerrs.Wrapf(err, "no such filter, name:%s", name)
			}
			s.Filters = append(s.Filters, filterFunc)
		}
	}
	// If there are multiple target services, weight cannot be empty
	if len(targetServiceList) > 1 && totalWeight == 0 {
		return errs.Newf(gerrs.ErrWrongConfig, "invalid target service configuration: %s",
			convert.ToJSONStr(targetServiceList))
	}
	return nil
}

// checkService verifies the upstream service configuration
func (r *FastHTTPRouter) checkService(clientMap map[string]*entity.BackendConfig, serviceName string) (*entity.BackendConfig, error) {
	if clientMap == nil {
		return nil, errs.New(gerrs.ErrWrongConfig, "empty client map")
	}
	// Check the completeness of the destination service configuration
	service, ok := clientMap[serviceName]
	if !ok {
		return nil, errs.Newf(gerrs.ErrWrongConfig, "no client config found for target service name: %s", serviceName)
	}
	// Check the completeness of the service configuration
	if service.Network == "" || service.Target == "" {
		return nil, errs.Newf(gerrs.ErrWrongConfig, "invalid service configuration: %s", convert.ToJSONStr(service))
	}
	// Validate the forwarding protocol
	if _, err := protocol.GetCliProtocolHandler(service.Protocol); err != nil {
		return nil, gerrs.Wrap(err, "invalid protocol")
	}
	return service, nil
}

// parsePluginConfig parses the plugin configuration
func (r *FastHTTPRouter) parsePluginConfig(pluginConfig *entity.Plugin) (interface{}, error) {
	// Set the default plugin type
	if pluginConfig.Type == "" {
		pluginConfig.Type = cplugin.DefaultType
	}
	pluginFactory := plugin.Get(pluginConfig.Type, pluginConfig.Name)
	if pluginFactory == nil {
		return nil, errs.Newf(gerrs.ErrWrongConfig,
			"invalid plugin type or name: %s", convert.ToJSONStr(pluginConfig))
	}
	// Assert as a gateway plugin
	gPlugin, ok := pluginFactory.(gwplugin.GatewayPlugin)
	if !ok {
		return nil, errs.Newf(gerrs.ErrWrongConfig,
			"plugin does not implement gateway plugin, name: %s", pluginConfig.Name)
	}
	decoder := &gwplugin.PropsDecoder{Props: pluginConfig.Props}
	if err := gPlugin.CheckConfig(pluginConfig.Name, decoder); err != nil {
		return nil, errs.Newf(gerrs.ErrWrongConfig,
			"check plugin configuration error: %s || config: %s", err, convert.ToJSONStr(pluginConfig))
	}
	return decoder.DecodedProps, nil
}

// mergePlugins retrieves the plugin list
// Plugin execution order: global plugins first, then service plugins, and finally router plugins
// Configuration priority: router plugin configuration > service plugin configuration > global plugin configuration
func (r *FastHTTPRouter) mergePlugins(routerPlugins, servicePlugins,
	globalPlugins []*entity.Plugin) []*entity.Plugin {
	// Put the plugins in the array in the order of router plugin configuration, service plugin configuration,
	// and global plugin configuration
	var plugins []*entity.Plugin
	plugins = append(plugins, globalPlugins...)
	plugins = append(plugins, servicePlugins...)
	plugins = append(plugins, routerPlugins...)

	// Reverse the order for deduplication, keeping the plugins with the smallest scope
	sort.SliceStable(plugins, func(i, j int) bool {
		return true
	})

	// Final list of effective plugins
	resultPlugins := make([]*entity.Plugin, 0, len(plugins))
	// Map to track duplicates
	m := make(map[string]struct{})
	for _, p := range plugins {
		if p.Disable {
			// Disable the plugin
			continue
		}
		if _, ok := m[p.Name]; !ok {
			m[p.Name] = struct{}{}
			resultPlugins = append(resultPlugins, p)
		}
	}

	// Reverse the order again to adjust the execution order
	sort.SliceStable(resultPlugins, func(i, j int) bool {
		return true
	})
	return resultPlugins
}

// GetMatchRouter Match the router
// 1. Exact match
// 2. Longest prefix match
// 3. Regular expression match
// 4. Fine-grained match
// 5. Gray calculation
func (r *FastHTTPRouter) GetMatchRouter(ctx context.Context) (*entity.TargetService, error) {
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return nil, errs.New(gerrs.ErrWrongContext, "invalid http context")
	}
	// Router matching: exact match, longest prefix match, regular expression match
	routerItemList, err := r.matchRouterItem(fctx)
	if err != nil {
		return nil, gerrs.Wrap(err, "match_router_item_err")
	}
	// Fine-grained match
	routerItem, err := r.getExactRouterItem(fctx, routerItemList)
	if err != nil {
		return nil, gerrs.Wrap(err, "get exact router err")
	}

	gwmsg.GwMessage(ctx).WithRouterID(routerItem.ID)

	// Rewrite the called interface to prevent explosion of interface dimensions in reporting
	if routerItem.ReportMethod {
		codec.Message(ctx).WithCallerMethod(routerItem.Method)
		codec.Message(ctx).WithCalleeMethod(routerItem.Method)
	}

	// Gray calculation
	targetService, err := r.getGreyServiceName(fctx, routerItem.HashKey, routerItem.TargetService)
	if err != nil {
		// This has been validated during configuration initialization, so this error should not occur
		return nil, gerrs.Wrap(err, "get_proxy_service_err")
	}
	// Rewrite path
	rewritePath := r.getRewritePath(fctx, routerItem, targetService)
	// Set the reported backend interface
	gwmsg.GwMessage(ctx).WithUpstreamMethod(r.getUpstreamMethod(string(fctx.Path()), rewritePath, routerItem,
		targetService))
	if rewritePath != "" {
		fctx.Request.URI().SetPath(rewritePath)
	}
	return targetService, nil
}

// Get the reported backend service interface
func (r *FastHTTPRouter) getUpstreamMethod(originPath, rewritePath string, routerItem *entity.RouterItem,
	targetService *entity.TargetService) string {
	if rewritePath != "" && !strings.HasPrefix(rewritePath, "/") {
		rewritePath = fmt.Sprintf("/%s", rewritePath)
	}
	// If report_method is false, report the actual requested interface
	if !routerItem.ReportMethod {
		if rewritePath != "" {
			return rewritePath
		}
		return originPath
	}
	// For interface paths that contain parameters, such as: /w/{article_id}, support reporting only the prefix /w/
	// to avoid explosion of interface dimensions in the monitoring system.
	// If report_method is true, only report the configured path, usually prefix matching
	routerRewrite := routerItem.ReWrite
	if targetService.ReWrite != "" {
		routerRewrite = targetService.ReWrite
	}
	if routerRewrite != "" {
		return routerRewrite
	}
	return routerItem.Method
}

// getRewritePath Get the rewritten path
func (r *FastHTTPRouter) getRewritePath(fctx *fasthttp.RequestCtx, routerItem *entity.RouterItem,
	targetService *entity.TargetService) string {
	if fctx == nil || routerItem == nil || targetService == nil {
		return ""
	}
	// Use the most granular rewrite configuration
	// Rewrite target-level path
	path := string(fctx.Path())
	// Remove prefix
	if targetService.StripPath || targetService.ReWrite != "" {
		return r.assemblePath(path, targetService.ReWrite, routerItem.Method, targetService.StripPath)
	}

	if routerItem.StripPath || routerItem.ReWrite != "" {
		return r.assemblePath(path, routerItem.ReWrite, routerItem.Method, routerItem.StripPath)
	}
	return ""
}

// Assemble the path
func (r *FastHTTPRouter) assemblePath(originPath, rewritePath, method string, stripPath bool) string {
	// If rewrite is an exact path, such as: /user/info, return directly
	if rewritePath != "" && !strings.HasSuffix(rewritePath, "/") {
		return rewritePath
	}

	// If rewrite is empty, it means removing the prefix, return directly
	if rewritePath == "" {
		return strings.TrimPrefix(originPath, method)
	}

	// If rewrite is not empty, it means /rewrite/
	// Concatenate after removing the prefix
	if stripPath {
		return fmt.Sprintf("%s%s", rewritePath, strings.TrimPrefix(originPath, method))
	}
	// Concatenate directly
	return fmt.Sprintf("%s%s", rewritePath, strings.TrimPrefix(originPath, "/"))
}

// matchRouterItem Match the router by path
// 1. First match with exact routes
// 2. Then match with longest prefix routes
// 3. Iterate through all regular expression routes
func (r *FastHTTPRouter) matchRouterItem(fctx *fasthttp.RequestCtx) ([]*entity.RouterItem, error) {
	path := string(fctx.Path())
	// Exact route matching
	if item, ok := r.getOpts().RadixTree.Get(path); ok {
		return item.([]*entity.RouterItem), nil
	}

	// Longest prefix route matching
	longestPrefix, item, ok := r.getOpts().RadixTree.LongestPrefix(path)
	if ok && longestPrefix != "/" && strings.HasSuffix(longestPrefix, "/") {
		return item.([]*entity.RouterItem), nil
	}

	// Regular expression route matching
	// Match in the order of configuration, no priority
	// The reason for iterating through regular expressions here is:
	// 1. Regular expression matching is very rare in route configurations, most of them are exact matches or longest
	//    prefix matches
	// 2. Regular expressions have been compiled, so the performance of regular expression matching is acceptable
	for _, item := range r.getOpts().RegRouterList {
		if item.MatchString(path) {
			return item.ItemList, nil
		}
	}

	return nil, errs.New(gerrs.ErrPathNotFound, "no router matched")
}

// getExactRouterItem After matching the route, perform fine-grained matching.
// The input parameter routerItemList usually has only about 2 items, so there is no performance issue with two
// iterations in the method.
// Matching logic:
//  1. First match the router item with a configured host; if no host is configured, match all router items.
//  2. Then match the rule, return the matched router item if there is a match; if no rule is matched, return the first
//     router item without a rule configured.
//  3. If no match is found, return an error.
func (r *FastHTTPRouter) getExactRouterItem(fctx *fasthttp.RequestCtx,
	routerItemList []*entity.RouterItem) (*entity.RouterItem, error) {
	host := string(fctx.Host())
	hostMatchList, err := r.getHostMatchRouterItemList(routerItemList, host)
	if err != nil {
		return nil, gerrs.Wrap(err, "get no host match route item")
	}

	// Get the router item that matches the rule
	routerItem, err := r.getRuleMatchItem(fctx, hostMatchList)
	if err != nil {
		return nil, gerrs.Wrap(err, "rule match err")
	}
	return routerItem, nil
}

// getHostMatchRouterItemList Get the list of routes that match the host
// Prioritize returning all router items configured with a host; if no match is found, return the router items without
// a configured host.
func (r *FastHTTPRouter) getHostMatchRouterItemList(routerItemList []*entity.RouterItem,
	host string) ([]*entity.RouterItem, error) {
	// Iterate and check the host
	// Match in the order of configuration, no priority
	var hostMatchList []*entity.RouterItem
	// Router items without a configured host
	var noHostRouterItemList []*entity.RouterItem
	// Get the items that match the host
	for _, item := range routerItemList {
		if len(item.HostMap) == 0 {
			noHostRouterItemList = append(noHostRouterItemList, item)
			continue
		}
		if _, ok := item.HostMap[host]; ok {
			hostMatchList = append(hostMatchList, item)
		}
	}

	// Prioritize matching the rules corresponding to the host, if there are no items with a host, then match all items
	if len(hostMatchList) != 0 {
		return hostMatchList, nil
	}
	// Use the router items without a configured host as a fallback
	if len(noHostRouterItemList) != 0 {
		return noHostRouterItemList, nil
	}

	return nil, errs.New(gerrs.ErrPathNotFound, "no host match router item found")
}

// getRuleMatchItem Get the rule-matched item
// Prioritize returning the item with a configured rule that matches; otherwise, return the item without a configured
// rule.
func (r *FastHTTPRouter) getRuleMatchItem(fctx *fasthttp.RequestCtx,
	secondMatchList []*entity.RouterItem) (*entity.RouterItem, error) {
	var noRuleItemList []*entity.RouterItem
	for _, item := range secondMatchList {
		if item.Rule == nil || len(item.Rule.Conditions) == 0 {
			noRuleItemList = append(noRuleItemList, item)
			continue
		}
		// Match according to the rule, return if matched
		matched, err := rule.MatchRule(fctx, item.Rule, DefaultGetString)
		if err != nil {
			log.ErrorContextf(fctx, "match rule err:%s", err)
			continue
		}
		if matched {
			return item, nil
		}
	}
	// If no rule is matched, return the item without a configured rule
	if len(noRuleItemList) > 0 {
		return noRuleItemList[0], nil
	}
	return nil, errs.New(gerrs.ErrPathNotFound, "get no rule matched router item")
}

// getGreyServiceName Get the target service name through the grey strategy
func (r *FastHTTPRouter) getGreyServiceName(fctx *fasthttp.RequestCtx,
	hashKey string, svrs []*entity.TargetService) (*entity.TargetService, error) {
	// Target service cannot be empty
	if len(svrs) == 0 {
		return nil, errs.New(gerrs.ErrTargetServiceNotFound, "empty dst services")
	}

	// Optimize calculation, no need to calculate weights if there is only one service
	if len(svrs) == 1 {
		return svrs[0], nil
	}

	sumWeight := 0
	for _, svr := range svrs {
		sumWeight += svr.Weight
	}
	// If all weights are zero, return an error. (This has been validated during configuration loading, but validate it
	// here as well)
	if sumWeight == 0 {
		return nil, errs.New(gerrs.ErrTargetServiceNotFound, "invalid svr weight")
	}
	rad := rand.Intn(sumWeight)
	// Check if there is a state-based grey strategy
	if hashKey != "" {
		val := DefaultGetString(fctx, hashKey)
		if val != "" {
			rad = int(convert.Fnv32(val) % uint32(sumWeight))
		}
	}

	// Custom weighted grey strategy
	total := 0
	for _, svr := range svrs {
		total += svr.Weight
		if rad < total {
			return svr, nil
		}
	}
	return nil, nil
}

// DefaultGetString Get the value of the request parameter. Business can override this method to customize.
var DefaultGetString rule.GetStringFunc = func(ctx context.Context, key string) string {
	fctx, ok := ctx.(*fasthttp.RequestCtx)
	if !ok {
		return ""
	}
	return http.GetString(fctx, key)
}
