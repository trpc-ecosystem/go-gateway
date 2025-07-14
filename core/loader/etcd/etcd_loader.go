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

// Package etcd provides a router config loader with etcd.
package etcd

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v3"
	etcd "trpc.group/trpc-go/trpc-config-etcd"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/core/config"
	"trpc.group/trpc-go/trpc-gateway/core/entity"
	"trpc.group/trpc-go/trpc-gateway/core/router"
	"trpc.group/trpc-go/trpc-gateway/internal/util"
	tconfig "trpc.group/trpc-go/trpc-go/config"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/metrics"
)

const (
	etdRouter = "etcd_router"

	// Router configuration key
	routerConfKey = "router_conf"
)

func init() {
	config.RegisterConfLoader(etdRouter, &ConfLoader{})
}

// ConfLoader is the router configuration loader with etcd.
type ConfLoader struct {
}

// LoadConf loads the configuration from etcd.
func (l *ConfLoader) LoadConf(ctx context.Context, protocol string) (err error) {
	conf, err := tconfig.GetString(routerConfKey)
	if err != nil {
		return gerrs.Wrap(err, "get router from etcd err")
	}

	log.Infof("router config:%s", conf)
	if len(conf) == 0 {
		// No configuration obtained, return
		return errs.New(gerrs.ErrWrongConfig, "get no router config")
	}
	conf = util.ExpandEnv(conf)
	var proxyConfig entity.ProxyConfig
	err = yaml.Unmarshal([]byte(conf), &proxyConfig)
	if err != nil {
		return gerrs.Wrap(err, "unmarshal_router_conf_err")
	}
	err = router.GetRouter(protocol).InitRouterConfig(ctx, &proxyConfig)
	if err != nil {
		return gerrs.Wrap(err, "load proxy config err")
	}

	// Watch for remote configuration changes
	c, err := tconfig.GlobalKV().(*etcd.Client).Watch(context.TODO(), routerConfKey)
	if err != nil {
		return gerrs.Wrap(err, "watch router config err")
	}
	go func() {
		for r := range c {
			log.Infof("event: %d, value: %s", r.Event(), r.Value())
			buf := []byte(util.ExpandEnv(r.Value()))
			if err := l.parseAndInit(buf, protocol); err != nil {
				log.Errorf("reload router conf failed: %s", err)
				l.reportErr(err)
			}
		}
	}()

	return
}

// reportErr reports errors for monitoring and alerting purposes
func (l *ConfLoader) reportErr(err error) {
	if err == nil {
		return
	}
	// 失败了上报metric，配置监控报警
	dims := []*metrics.Dimension{
		{
			Name:  "err_code",
			Value: fmt.Sprint(errs.Code(err)),
		},
		{
			Name:  "err_msg",
			Value: err.Error(),
		},
	}
	indices := []*metrics.Metrics{
		metrics.NewMetrics("reload_router_err_count", float64(1), metrics.PolicySUM),
	}
	err = metrics.Report(metrics.NewMultiDimensionMetricsX(gerrs.GatewayERRKey, dims, indices))
	if err != nil {
		log.Errorf("report reload err count failed:%s", err)
	}
}

// parseAndInit parses and initializes the configuration
func (l *ConfLoader) parseAndInit(buf []byte, protocol string) error {
	var proxyConfig entity.ProxyConfig
	if err := yaml.Unmarshal(buf, &proxyConfig); err != nil {
		return gerrs.Wrap(err, "unmarshal proxy config err")
	}

	if err := router.GetRouter(protocol).InitRouterConfig(context.Background(), &proxyConfig); err != nil {
		return gerrs.Wrap(err, "init router err")
	}
	return nil
}
