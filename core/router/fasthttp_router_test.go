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

package router

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"gopkg.in/yaml.v3"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/core/entity"
	"trpc.group/trpc-go/trpc-gateway/core/rule"
	cprotocol "trpc.group/trpc-go/trpc-gateway/core/service/protocol"
	"trpc.group/trpc-go/trpc-gateway/core/service/protocol/mock"
	mockplugin "trpc.group/trpc-go/trpc-gateway/plugin/mock"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/plugin"
)

func TestFastHTTPRouter_getString(t *testing.T) {
	key := DefaultGetString(context.Background(), "suid")
	assert.Equal(t, "", key)
	fCtx := &fasthttp.RequestCtx{}
	fCtx.Request.Header.Set("suid", "123")
	key = DefaultGetString(fCtx, "suid")
	assert.Equal(t, "123", key)
}

func getTestProxyConfig(t *testing.T) *entity.ProxyConfig {
	// load config
	confBytes, err := os.ReadFile("../../testdata/router.yaml")
	assert.Nil(t, err)
	var proxyConfig entity.ProxyConfig
	err = yaml.Unmarshal(confBytes, &proxyConfig)
	assert.Nil(t, err)
	return &proxyConfig
}

func TestFastHTTPRouter_InitRouterConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockGWPlugin := mockplugin.NewMockGatewayPlugin(ctrl)

	mockGWPlugin.EXPECT().Setup(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockGWPlugin.EXPECT().Type().Return("gateway").AnyTimes()
	mockGWPlugin.EXPECT().CheckConfig(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockServerFilter := func(ctx context.Context, req interface{},
		next filter.ServerHandleFunc) (rsp interface{}, err error) {
		return nil, nil
	}

	cprotocol.RegisterCliProtocolHandler("fasthttp", mock.NewMockCliProtocolHandler(ctrl))
	// Initialize mock plugin
	plugin.Register("proxyinfo", mockGWPlugin)
	filter.Register("proxyinfo", mockServerFilter, nil)
	plugin.Register("tnewsauth", mockGWPlugin)
	filter.Register("tnewsauth", mockServerFilter, nil)
	plugin.Register("tnewswebauth", mockGWPlugin)
	filter.Register("tnewswebauth", mockServerFilter, nil)
	plugin.Register("auth", mockGWPlugin)
	filter.Register("auth", mockServerFilter, nil)
	r := NewFastHTTPRouter()
	err := r.InitRouterConfig(context.Background(), &entity.ProxyConfig{})
	assert.NotNil(t, err)
	// Initialize plugin
	// Initialize config
	proxyConfig := getTestProxyConfig(t)

	r = NewFastHTTPRouter()
	err = r.InitRouterConfig(context.Background(), proxyConfig)
	assert.Nil(t, err)
	radixMap := r.getOpts().RadixTree.ToMap()
	userRouterList := radixMap["/user/info"]
	assert.NotNil(t, userRouterList)
	userRouterItemList, ok := userRouterList.([]*entity.RouterItem)
	assert.Equal(t, true, ok)
	// Router list is greater than 0
	assert.Greater(t, len(userRouterItemList), 0)
	// Router plugins is greater than 0
	assert.Greater(t, len(userRouterItemList[0].Plugins), 0)
	// Router rule is not nil
	assert.NotNil(t, userRouterItemList[0].Rule)
	// Router rule conditions is not empty
	assert.Greater(t, len(userRouterItemList[0].Rule.Conditions), 0)
	// Target service is greater than 0
	assert.Greater(t, len(userRouterItemList[0].TargetService), 0)
	// Target service plugins is greater than 0
	assert.Greater(t, len(userRouterItemList[0].TargetService[0].Plugins), 0)
	// Target service plugin order
	var plugins []string
	for _, p := range userRouterItemList[0].TargetService[0].Plugins {
		plugins = append(plugins, p.Name)
	}
	assert.ElementsMatch(t, plugins, []string{"proxyinfo", "auth", "tnewsauth"})
	c, _ := yaml.Marshal(radixMap)
	t.Log(string(c))
	// Target service is empty
	tmpService := *proxyConfig.Router[0].TargetService[0]
	proxyConfig.Router[0].TargetService[0].Service = "xxxx"
	err = r.InitRouterConfig(context.Background(), proxyConfig)
	assert.NotNil(t, err)
	proxyConfig.Router[0].TargetService[0].Service = tmpService.Service
	// Target service configuration error
	tmpClient := *proxyConfig.Client[0]
	proxyConfig.Client[0].Protocol = ""
	err = r.InitRouterConfig(context.Background(), proxyConfig)
	assert.NotNil(t, err)
	proxyConfig.Client[0].Protocol = tmpClient.Protocol
	// Item method is empty
	tmpRouter := *proxyConfig.Router[0]
	proxyConfig.Router[0].Method = ""
	err = r.InitRouterConfig(context.Background(), proxyConfig)
	assert.NotNil(t, err)
	proxyConfig.Router[0].Method = tmpRouter.Method

	// Item method is "/"
	tmpRouter = *proxyConfig.Router[0]
	proxyConfig.Router[0].Method = "/"
	err = r.InitRouterConfig(context.Background(), proxyConfig)
	assert.NotNil(t, err)
	proxyConfig.Router[0].Method = tmpRouter.Method
	// Target service weight configuration error
	tmpTargetService := proxyConfig.Router[0].TargetService
	proxyConfig.Router[0].TargetService = []*entity.TargetService{
		{
			Service: "trpc.inews.user.User",
			Weight:  0,
		},
		{
			Service: "trpc.inews.user.UserV2",
			Weight:  0,
		},
	}
	err = r.InitRouterConfig(context.Background(), proxyConfig)
	assert.NotNil(t, err)
	proxyConfig.Router[0].TargetService = tmpTargetService

	// Target service configuration is empty
	proxyConfig.Router[0].TargetService = []*entity.TargetService{}
	err = r.InitRouterConfig(context.Background(), proxyConfig)
	assert.NotNil(t, err)
	proxyConfig.Router[0].TargetService = tmpTargetService

	// Failed to get plugin
	tmpPlugin := proxyConfig.Router[0].Plugins
	proxyConfig.Router[0].Plugins = []*entity.Plugin{
		{
			Name:  "no_auth",
			Type:  "",
			Props: nil,
		},
	}
	err = r.InitRouterConfig(context.Background(), proxyConfig)
	assert.NotNil(t, err)
	proxyConfig.Router[0].Plugins = tmpPlugin

	// Plugin type assertion failed
	mockTrpcPlugin := mockplugin.NewMockFactory(ctrl)
	mockTrpcPlugin.EXPECT().Type().Return("gateway").AnyTimes()
	plugin.Register("auth", mockTrpcPlugin)
	err = r.InitRouterConfig(context.Background(), proxyConfig)
	assert.NotNil(t, err)
	t.Log(err)
	plugin.Register("auth", mockGWPlugin)

	// Plugin configuration validation failed
	// Reset ctrl
	ctrl2 := gomock.NewController(t)
	defer ctrl2.Finish()
	mockGWPlugin = mockplugin.NewMockGatewayPlugin(ctrl2)
	mockGWPlugin.EXPECT().Setup(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockGWPlugin.EXPECT().Type().Return("gateway").AnyTimes()
	mockGWPlugin.EXPECT().CheckConfig(gomock.Any(), gomock.Any()).Return(errors.New("err")).AnyTimes()
	// Initialize mock plugins
	plugin.Register("proxyinfo", mockGWPlugin)
	plugin.Register("tnewsauth", mockGWPlugin)
	plugin.Register("tnewswebauth", mockGWPlugin)
	plugin.Register("auth", mockGWPlugin)

	err = r.InitRouterConfig(context.Background(), proxyConfig)
	assert.NotNil(t, err)
	t.Log(err)
	mockGWPlugin.EXPECT().CheckConfig(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	_, err = NewFastHTTPRouter().checkService(nil, "demo")
	assert.NotNil(t, err)
}

func TestFastHTTPRouter_GetMatchRouter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockGWPlugin := mockplugin.NewMockGatewayPlugin(ctrl)

	mockGWPlugin.EXPECT().Setup(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockGWPlugin.EXPECT().Type().Return("gateway").AnyTimes()
	mockGWPlugin.EXPECT().CheckConfig(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	// Initialize mock plugins
	mockServerFilter := func(ctx context.Context, req interface{},
		next filter.ServerHandleFunc) (rsp interface{}, err error) {
		return nil, nil
	}
	plugin.Register("proxyinfo", mockGWPlugin)
	filter.Register("proxyinfo", mockServerFilter, nil)
	plugin.Register("tnewsauth", mockGWPlugin)
	filter.Register("tnewsauth", mockServerFilter, nil)
	plugin.Register("tnewswebauth", mockGWPlugin)
	filter.Register("tnewswebauth", mockServerFilter, nil)
	plugin.Register("auth", mockGWPlugin)
	filter.Register("auth", mockServerFilter, nil)

	// Load configuration
	proxyConfig := getTestProxyConfig(t)
	r := NewFastHTTPRouter()
	err := r.InitRouterConfig(context.Background(), proxyConfig)
	assert.Nil(t, err)
	// Incorrect ctx type
	_, err = r.GetMatchRouter(context.Background())
	assert.Equal(t, errs.Code(err), gerrs.ErrWrongContext)
	ctx, _ := gwmsg.WithNewGWMessage(context.Background())
	// Matching failed
	fCtx := &fasthttp.RequestCtx{}
	fCtx.Request.SetRequestURI("/no/match/uri")
	ctx = http.WithRequestContext(ctx, fCtx)
	_, err = r.GetMatchRouter(ctx)
	assert.NotNil(t, err)

	// Exact match success
	ctx, _ = gwmsg.WithNewGWMessage(context.Background())
	fCtx = &fasthttp.RequestCtx{}
	fCtx.Request.SetRequestURI("/user/info")
	ctx = http.WithRequestContext(ctx, fCtx)
	targetService, err := r.GetMatchRouter(ctx)
	assert.Nil(t, err)
	assert.NotEmpty(t, targetService.BackendConfig.ServiceName)
	assert.Greater(t, len(targetService.Plugins), 0)
	assert.Equal(t, "/user/infoV2", gwmsg.GwMessage(ctx).UpstreamMethod())

	// If the leaf node does not exist, add it
	routerList := []*entity.RouterItem{
		{
			Method:        "/usr/info/ext",
			TargetService: nil,
		},
	}
	r.opts.RadixTree.Insert("/usr/info/ext", routerList)

	fCtx = &fasthttp.RequestCtx{}
	fCtx.Request.SetRequestURI("/usr/info/ext")
	ctx = http.WithRequestContext(context.Background(), fCtx)
	_, err = r.GetMatchRouter(ctx)
	assert.NotNil(t, err)

	// Prefix match with strip_path
	ctx, _ = gwmsg.WithNewGWMessage(context.Background())
	fCtx = &fasthttp.RequestCtx{}
	fCtx.Request.SetRequestURI("/user/other")
	ctx = http.WithRequestContext(ctx, fCtx)
	_, err = r.GetMatchRouter(ctx)
	assert.Nil(t, err)
	assert.Equal(t, "/other", string(fCtx.Path()))
	assert.Equal(t, "/other", gwmsg.GwMessage(ctx).UpstreamMethod())

	routerList[0].TargetService = []*entity.TargetService{
		{},
	}
	routerList[0].Rule = &entity.RuleItem{
		Conditions: []*entity.Condition{
			{
				Key:  "devid",
				Val:  "xxx",
				Oper: "==",
			},
		},
		Expression:       "0&&4",
		ConditionIdxList: []int{4},
		OptList:          nil,
	}

	fCtx = &fasthttp.RequestCtx{}
	fCtx.Request.SetRequestURI("/usr/info/ext")
	ctx = http.WithRequestContext(context.Background(), fCtx)
	_, err = r.GetMatchRouter(ctx)
	assert.NotNil(t, err)
	// Exact match failed
	fCtx = &fasthttp.RequestCtx{}
	fCtx.Request.SetRequestURI("/user/info")
	fCtx.Request.SetHost("r.inews.qq.com")
	ctx = http.WithRequestContext(context.Background(), fCtx)
	_, err = r.GetMatchRouter(ctx)
	assert.NotNil(t, err)

	// Normal match
	ctx, _ = gwmsg.WithNewGWMessage(context.Background())
	fCtx = &fasthttp.RequestCtx{}
	fCtx.Request.SetRequestURI("/user/info4")
	fCtx.Request.SetHost("r.inews.qq.com")
	ctx = http.WithRequestContext(ctx, fCtx)
	_, err = r.GetMatchRouter(ctx)
	assert.Nil(t, err)
	assert.Equal(t, "/user/info4", string(fCtx.Path()))
	assert.Equal(t, "/user/info4", gwmsg.GwMessage(ctx).UpstreamMethod())

	// Normal match with prefix reporting
	ctx, _ = gwmsg.WithNewGWMessage(context.Background())
	fCtx = &fasthttp.RequestCtx{}
	fCtx.Request.SetRequestURI("/user_prefix/info5")
	fCtx.Request.SetHost("r.inews.qq.com")
	ctx = http.WithRequestContext(ctx, fCtx)
	_, err = r.GetMatchRouter(ctx)
	assert.Nil(t, err)
	assert.Equal(t, "/user_prefix/", gwmsg.GwMessage(ctx).UpstreamMethod())
}

func Benchmark_matchRouterItem(b *testing.B) {
	// Load configuration
	confBytes, err := os.ReadFile("../../testdata/routerbenchmark.yaml")
	assert.Nil(b, err)
	var proxyConfig entity.ProxyConfig
	err = yaml.Unmarshal(confBytes, &proxyConfig)
	assert.Nil(b, err)
	r := NewFastHTTPRouter()
	err = r.InitRouterConfig(context.Background(), &proxyConfig)
	assert.Nil(b, err)
	for i := 0; i < b.N; i++ {
		// Exact match
		fCtx := &fasthttp.RequestCtx{}
		fCtx.Request.SetRequestURI("/invalid")
		_, err = r.matchRouterItem(fCtx)
		assert.NotNil(b, err)
	}
}

func TestFastHTTPRouter_matchRouterItem(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockGWPlugin := mockplugin.NewMockGatewayPlugin(ctrl)

	mockGWPlugin.EXPECT().Setup(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockGWPlugin.EXPECT().Type().Return("gateway").AnyTimes()
	mockGWPlugin.EXPECT().CheckConfig(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	// Initialize mock plugins
	plugin.Register("proxyinfo", mockGWPlugin)
	plugin.Register("tnewsauth", mockGWPlugin)
	plugin.Register("tnewswebauth", mockGWPlugin)
	plugin.Register("auth", mockGWPlugin)
	// Load configuration
	proxyConfig := getTestProxyConfig(t)
	r := NewFastHTTPRouter()
	err := r.InitRouterConfig(context.Background(), proxyConfig)
	assert.Nil(t, err)
	// Exact match
	fCtx := &fasthttp.RequestCtx{}
	fCtx.Request.SetRequestURI("/user/info")
	routerItemList, err := r.matchRouterItem(fCtx)
	assert.Nil(t, err)
	assert.Greater(t, len(routerItemList), 0)

	// Longest prefix match
	fCtx = &fasthttp.RequestCtx{}
	fCtx.Request.SetRequestURI("/user/ext_info")
	routerItemList, err = r.matchRouterItem(fCtx)
	assert.Nil(t, err)
	assert.Greater(t, len(routerItemList), 0)

	// Regular expression match
	fCtx = &fasthttp.RequestCtx{}
	fCtx.Request.SetRequestURI("/feed/flow")
	routerItemList, err = r.matchRouterItem(fCtx)
	assert.Nil(t, err)
	assert.Greater(t, len(routerItemList), 0)

	// Match failed
	fCtx = &fasthttp.RequestCtx{}
	fCtx.Request.SetRequestURI("/article/info")
	routerItemList, err = r.matchRouterItem(fCtx)
	assert.Equal(t, errs.Code(err), gerrs.ErrPathNotFound)
	assert.Nil(t, routerItemList)
}

func TestFastHTTPRouter_getExactRouterItem(t *testing.T) {
	// Load configuration
	proxyConfig := getTestProxyConfig(t)
	r := NewFastHTTPRouter()
	err := r.InitRouterConfig(context.Background(), proxyConfig)
	assert.Nil(t, err)

	routerItemList := []*entity.RouterItem{
		{
			Method: "/user/info",
			Host:   []string{"r.inews.qq.com"},
			HostMap: map[string]struct{}{
				"r.inews.qq.com": {},
			},
		},
		{
			Method: "/user/info",
			Host:   []string{"r.inews.qq.com"},
			HostMap: map[string]struct{}{
				"r.inews.qq.com": {},
			},
			Rule: &entity.RuleItem{
				Conditions: []*entity.Condition{
					{
						Key:  "devid",
						Val:  "xxx",
						Oper: "==",
					},
				},
				Expression: "0",
			},
		},
		{
			Method:  "/user/info",
			Host:    []string{},
			HostMap: map[string]struct{}{},
		},
		{
			Method: "/user/info",
			Host:   []string{"qq.com"},
			HostMap: map[string]struct{}{
				"qq.com": {},
			},
			Rule: &entity.RuleItem{
				Conditions: []*entity.Condition{
					{
						Key:  "devid",
						Val:  "xxx",
						Oper: "==",
					},
				},
				Expression: "0",
			},
		},
		{
			Method: "/user/info",
			Host:   []string{"qq.com"},
			HostMap: map[string]struct{}{
				"qq.com": {},
			},
			Rule: &entity.RuleItem{
				Conditions: []*entity.Condition{
					{
						Key:  "devid",
						Val:  "xxx",
						Oper: "==",
					},
				},
				Expression: "",
			},
		},
		{
			Method: "/user/",
			Host:   []string{"qq.com"},
			HostMap: map[string]struct{}{
				"qq.com": {},
			},
			Rule: &entity.RuleItem{
				Conditions: []*entity.Condition{
					{
						Key:  "devid",
						Val:  "xxx",
						Oper: "==",
					},
				},
				Expression: "",
			},
		},
	}
	// Iterate and initialize rules
	for _, item := range routerItemList {
		if item.Rule != nil && item.Rule.Expression != "" {
			err = rule.FormatRule(item.Rule)
			assert.Nil(t, err)
		}
	}
	// Matched by host, without rule
	fCtx := &fasthttp.RequestCtx{}
	fCtx.Request.SetRequestURI("/user/info")
	fCtx.Request.SetHost("r.inews.qq.com")
	routerItem, err := r.getExactRouterItem(fCtx, routerItemList)
	assert.Nil(t, err)
	assert.Nil(t, routerItem.Rule)

	// Matched by host and rule
	fCtx = &fasthttp.RequestCtx{}
	fCtx.Request.SetRequestURI("/user/info")
	fCtx.Request.SetHost("r.inews.qq.com")
	fCtx.Request.Header.Set("devid", "xxx")
	routerItem, err = r.getExactRouterItem(fCtx, routerItemList)
	assert.Nil(t, err)
	assert.NotNil(t, routerItem.Rule)
	assert.Equal(t, routerItem.Rule.Conditions[0].Val, "xxx")

	// Matched by host but no matching rule
	fCtx = &fasthttp.RequestCtx{}
	fCtx.Request.SetRequestURI("/user/info")
	fCtx.Request.SetHost("qq.com")
	fCtx.Request.Header.Set("devid", "yyy")
	_, err = r.getExactRouterItem(fCtx, routerItemList)
	assert.Equal(t, errs.Code(err), gerrs.ErrPathNotFound)

	// No matching host
	fCtx = &fasthttp.RequestCtx{}
	fCtx.Request.SetRequestURI("/user/info")
	fCtx.Request.SetHost("w.inews.qq.com")
	routerItem, err = r.getExactRouterItem(fCtx, routerItemList)
	assert.Nil(t, err)
	assert.Equal(t, routerItem.Host, []string{})

	// Host matching failed
	routerItemList = []*entity.RouterItem{
		{
			Method: "/user/info",
			Host:   []string{"r.inews.qq.com"},
			HostMap: map[string]struct{}{
				"r.inews.qq.com": {},
			},
			TargetService: nil,
			HashKey:       "",
			ReWrite:       "",
			Plugins:       nil,
		},
	}
	fCtx = &fasthttp.RequestCtx{}
	fCtx.Request.SetRequestURI("/user/info")
	fCtx.Request.SetHost("w.inews.qq.com")
	_, err = r.getExactRouterItem(fCtx, routerItemList)
	assert.NotNil(t, err)
}

func TestFastHTTPRouter_getGreyServiceName(t *testing.T) {
	router := DefaultFastHTTPRouter
	ctx := &fasthttp.RequestCtx{}
	// Empty service
	var svrs []*entity.TargetService
	target, err := router.getGreyServiceName(ctx, "", svrs)
	assert.Equal(t, gerrs.ErrTargetServiceNotFound, errs.Code(err))
	assert.Nil(t, target)

	// One service
	svrs = []*entity.TargetService{
		{
			Service: "target.service.a",
			Weight:  0,
		},
	}
	target, err = router.getGreyServiceName(ctx, "", svrs)
	assert.Nil(t, err)
	assert.Equal(t, "target.service.a", target.Service)

	// Two services
	svrs = []*entity.TargetService{
		{
			Service: "target.service.a",
			Weight:  1,
		},
		{
			Service: "target.service.a",
			Weight:  0,
		},
	}
	target, err = router.getGreyServiceName(ctx, "", svrs)
	assert.Nil(t, err)
	assert.Equal(t, "target.service.a", target.Service)

	// Two services with empty weights
	svrs = []*entity.TargetService{
		{
			Service: "target.service.a",
			Weight:  0,
		},
		{
			Service: "target.service.b",
			Weight:  0,
		},
	}
	target, err = router.getGreyServiceName(ctx, "", svrs)
	assert.Equal(t, gerrs.ErrTargetServiceNotFound, errs.Code(err))
	assert.Nil(t, target)

	// Two services with non-empty hash key
	ctx.Request.Header.Set("suid", "12345")
	svrs = []*entity.TargetService{
		{
			Service: "target.service.a",
			Weight:  1,
		},
		{
			Service: "target.service.b",
			Weight:  1,
		},
	}
	target, err = router.getGreyServiceName(ctx, "suid", svrs)
	assert.Nil(t, err)
	assert.Equal(t, "target.service.a", target.Service)
}

func TestFastHTTPRouter_getRewritePath(t *testing.T) {
	v2Path := "/v2/"
	v1Path := "/v1/"
	v3Path := "/v3/"
	ctx := &fasthttp.RequestCtx{}
	ctx.URI().SetPath("/v1/user")
	r := &FastHTTPRouter{}
	rewrite := r.getRewritePath(ctx, nil, nil)
	assert.Equal(t, "", rewrite)

	routerItem := &entity.RouterItem{}
	targetService := &entity.TargetService{}
	rewrite = r.getRewritePath(ctx, routerItem, targetService)
	assert.Equal(t, "", rewrite)

	targetService.ReWrite = "/target_rewrite"
	rewrite = r.getRewritePath(ctx, routerItem, targetService)
	assert.Equal(t, "/target_rewrite", rewrite)

	targetService.ReWrite = v2Path
	targetService.StripPath = true
	routerItem.Method = v1Path
	rewrite = r.getRewritePath(ctx, routerItem, targetService)
	assert.Equal(t, "/v2/user", rewrite)

	targetService.ReWrite = v2Path
	targetService.StripPath = false
	routerItem.Method = v1Path
	rewrite = r.getRewritePath(ctx, routerItem, targetService)
	assert.Equal(t, "/v2/v1/user", rewrite)

	targetService.ReWrite = ""
	targetService.StripPath = true
	routerItem.Method = v1Path
	rewrite = r.getRewritePath(ctx, routerItem, targetService)
	assert.Equal(t, "user", rewrite)

	targetService.StripPath = false
	routerItem.ReWrite = "/router_rewrite"
	rewrite = r.getRewritePath(ctx, routerItem, targetService)
	assert.Equal(t, "/router_rewrite", rewrite)
	routerItem.ReWrite = ""

	targetService.StripPath = false
	routerItem.StripPath = true
	ctx.URI().SetPath("/v2/user")
	routerItem.Method = v2Path
	rewrite = r.getRewritePath(ctx, routerItem, targetService)
	assert.Equal(t, "user", rewrite)

	ctx.URI().SetPath("/v2/user")
	targetService.StripPath = false
	routerItem.StripPath = true
	routerItem.Method = v2Path
	routerItem.ReWrite = v3Path
	rewrite = r.getRewritePath(ctx, routerItem, targetService)
	assert.Equal(t, "/v3/user", rewrite)

	ctx.URI().SetPath("/v2/user")
	targetService.StripPath = false
	routerItem.StripPath = false
	routerItem.Method = v2Path
	routerItem.ReWrite = v3Path
	rewrite = r.getRewritePath(ctx, routerItem, targetService)
	assert.Equal(t, "/v3/v2/user", rewrite)
}

func TestFastHTTPRouter_mergePlugins(t *testing.T) {
	r := &FastHTTPRouter{}
	routerPlugins := []*entity.Plugin{
		{
			Name:  "a",
			Type:  "",
			Props: "router_config",
		},
		{
			Name: "b",
		},
	}
	servicePlugins := []*entity.Plugin{
		{
			Name:  "a",
			Props: "service_config",
		},
		{
			Name: "c",
		},
		{
			Name: "d",
		},
	}
	globalPlugins := []*entity.Plugin{
		{
			Name:  "a",
			Props: "global_config",
		},
		{
			Name: "e",
		},
		{
			Name: "f",
		},
	}
	resultPluginsList := r.mergePlugins(routerPlugins, servicePlugins, globalPlugins)
	assert.ElementsMatch(t, resultPluginsList, []*entity.Plugin{
		{
			Name: "e",
		},
		{
			Name: "f",
		},
		{
			Name: "c",
		},
		{
			Name: "d",
		},
		{
			Name:  "a",
			Type:  "",
			Props: "router_config",
		},
		{
			Name: "b",
		},
	})
}

func TestFastHTTPRouter_getUpstreamMethod(t *testing.T) {
	r := &FastHTTPRouter{}
	routerItem := &entity.RouterItem{}
	targetService := &entity.TargetService{}

	// Report full path, report the rewritten interface
	routerItem.ReportMethod = false
	upstreamMethod := r.getUpstreamMethod("/v1/user", "v2/user", routerItem, targetService)
	assert.Equal(t, "/v2/user", upstreamMethod)

	// Report prefix, report the configured interface
	routerItem.ReportMethod = true
	routerItem.Method = "/v0"
	routerItem.Method = "/v1"
	targetService.ReWrite = "/v2/"
	upstreamMethod = r.getUpstreamMethod("/v1/user", "v2/user", routerItem, targetService)
	assert.Equal(t, "/v2/", upstreamMethod)
}
