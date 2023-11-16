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

// Package entity defines entities related to route configuration.
package entity

import (
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/filter"
)

// ProxyConfig is the configuration options for the proxy.
type ProxyConfig struct {
	Router  []*RouterItem    `yaml:"router,omitempty" json:"router,omitempty"`
	Client  []*BackendConfig `yaml:"client,omitempty" json:"client,omitempty"`
	Plugins []*Plugin        `yaml:"plugins,omitempty" json:"plugins,omitempty"`
}

// Condition is a condition for routing rule.
type Condition struct {
	Key  string `yaml:"key,omitempty" json:"key,omitempty"`
	Val  string `yaml:"val,omitempty" json:"val,omitempty"`
	Oper string `yaml:"oper,omitempty" json:"oper,omitempty"`
	// ParsedVal is the parsed value of Val.
	ParsedVal interface{} `yaml:"-" json:"-"`
}

// RuleItem is a rule for routing.
type RuleItem struct {
	// Conditions are the conditions for routing rule.
	Conditions []*Condition `yaml:"conditions,omitempty" json:"conditions,omitempty"`
	// Expression is the expression for routing rule.
	Expression string `yaml:"expression,omitempty" json:"expression,omitempty"`
	// ConditionIdxList are the indexes of conditions for routing rule,parsed from Expression.
	ConditionIdxList []int `yaml:"-" json:"-"`
	// OptList are the options for routing rule,parsed from Expression, like || && etc.
	OptList []string `yaml:"-" json:"-"`
}

// TargetService is upstream service config.
type TargetService struct {
	// Service is the upstream service name.
	Service string `yaml:"service,omitempty" json:"service,omitempty"`
	// BackendConfig is the config for upstream service,corresponds to Service.
	BackendConfig *client.BackendConfig `yaml:"-" json:"-"`
	// Weight is the weight of upstream service.
	Weight int `yaml:"weight,omitempty" json:"weight,omitempty"`
	// ReWrite is the rewrite method for upstream service.
	ReWrite string `yaml:"rewrite,omitempty"`
	// StripPath define whether to strip the path prefix.
	// for example: if a request matches the route '/api/' with StripPath set to true, then forward it to '/user/info'
	StripPath bool `yaml:"strip_path,omitempty"`
	// Plugins include all plugin at the global, service, and router levels.
	Plugins []*Plugin `yaml:"-" json:"-"`
	// Filters include all filter function at the global, service, and router levels.
	Filters []filter.ServerFilter `yaml:"-" json:"-"`
}

// Plugin is gateway plugin config.
type Plugin struct {
	// Name 插件名称
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
	// 插件类型，默认为 gateway
	Type string `yaml:"type,omitempty" json:"type,omitempty"`
	// Props 插件属性，在插件逻辑里解析配置字段
	Props interface{} `yaml:"props,omitempty" json:"props,omitempty"`
	// 是否禁用
	Disable bool `yaml:"disable,omitempty" json:"disable,omitempty"`
}

// RouterItem is router config.
type RouterItem struct {
	// ID is the unique id of router.
	ID string `yaml:"id,omitempty" json:"id,omitempty"`
	// Method is the request path to match.
	Method string `yaml:"method,omitempty" json:"method,omitempty"`
	// Host is the request host list to match.
	Host []string `yaml:"host,omitempty" json:"host,omitempty"`
	// HostMap is parsed from Host, and used to match the request host.
	HostMap map[string]struct{} `yaml:"-" json:"-"`
	// IsRegexp defines whether the method is a regular expression.
	IsRegexp bool `yaml:"is_regexp,omitempty" json:"is_regexp,omitempty"`
	// Rule define the matching rules witch is used to match the request params
	Rule *RuleItem `yaml:"rule,omitempty" json:"rule,omitempty"`
	// TargetService is the target service list.
	TargetService []*TargetService `yaml:"target_service,omitempty" json:"targetService,omitempty"`
	// HashKey enables stateful gray-scale forwarding, such as devid, etc., and is optional.
	HashKey string `yaml:"hash_key,omitempty" json:"hash_key,omitempty"`
	// ReWrite redefines the interface path.
	ReWrite string `yaml:"rewrite,omitempty" json:"rewrite,omitempty"`
	// StripPath with true, the prefix is remove from the path
	// For example, when the request /api/user/info matches the route /api and rewrite is enabled,
	// it will be forwarded as /user/info. Please note that the rewrite rule takes precedence over strip_path.
	StripPath bool `yaml:"strip_path,omitempty" json:"strip_path,omitempty"`
	// ReportMethod only reports the HTTP method without reporting the path.
	// This is done to prevent explosion of monitoring dimensions in the called interface,
	// especially for interfaces like /a/{article_id}.
	ReportMethod bool `yaml:"report_method,omitempty" json:"report_method,omitempty"`
	// Plugins List of plugins
	Plugins []*Plugin `yaml:"plugins,omitempty" json:"plugins,omitempty"`
}

// BackendConfig refers to the configuration of the upstream service.
type BackendConfig struct {
	// BackendConfig is used to configure the upstream service in trpc.
	// The "inline" tag will flatten the fields of BackendConfig during parsing.
	client.BackendConfig `yaml:",inline"`
	// Plugins are list of service plugins.
	Plugins []*Plugin `yaml:"plugins,omitempty" json:"plugins,omitempty"`
}
