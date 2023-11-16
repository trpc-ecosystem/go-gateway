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

// Package plugin defines gateway plugins.
package plugin

import (
	"errors"
	"reflect"

	"gopkg.in/yaml.v3"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-go/plugin"
)

//go:generate mockgen -destination ./mock/gatewayplugin.go . GatewayPlugin
//go:generate mockgen -destination ./mock/trpcplugin.go trpc.group/trpc-go/trpc-go/plugin Factory
//go:generate mockgen -destination ./mock/decoder.go trpc.group/trpc-go/trpc-go/plugin Decoder

// GatewayPlugin 网关插件
type GatewayPlugin interface {
	// Factory embed tRPC plugin
	plugin.Factory

	// CheckConfig 网关插件配置校验，并进行默认值的初始化
	CheckConfig(name string, dec plugin.Decoder) error
}

// PropsDecoder 解析插件配置
type PropsDecoder struct {
	// 原始 props，类型为 map[string]interface{}
	Props interface{}
	// decode 之后的 props，类型为插件里定义的
	DecodedProps interface{}
}

// Decode 解析插件配置
func (d *PropsDecoder) Decode(cfg interface{}) error {
	if d.Props == nil {
		d.Props = make(map[string]interface{})
	}
	// 判断，需要是指针类型
	if reflect.ValueOf(cfg).Kind() != reflect.Ptr {
		return errors.New("need pointer cfg")
	}
	props, err := yaml.Marshal(d.Props)
	if err != nil {
		return gerrs.Wrap(err, "marshal plugin props err")
	}
	if err = yaml.Unmarshal(props, cfg); err != nil {
		return gerrs.Wrap(err, "unmarshal plugin props err")
	}
	d.DecodedProps = cfg
	return nil
}
