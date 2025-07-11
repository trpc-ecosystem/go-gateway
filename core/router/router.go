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

// Package router Gateway routing module
package router

import (
	"context"
	"regexp"
	"sync"

	"github.com/armon/go-radix"
	"trpc.group/trpc-go/trpc-gateway/core/entity"
)

const (
	protocolFastHTTP = "fasthttp"
)

var (
	routers   = make(map[string]Router)
	muxRouter sync.RWMutex
)

// RegisterRouter registers the protocol router selector
func RegisterRouter(protocol string, r Router) {
	muxRouter.Lock()
	routers[protocol] = r
	muxRouter.Unlock()
}

// GetRouter gets the router selector based on the protocol
var GetRouter = func(protocol string) Router {
	muxRouter.RLock()
	r := routers[protocol]
	muxRouter.RUnlock()
	return r
}

//go:generate mockgen -destination=./mock/mockrouter.go  . Router

// Router defines the router interface
type Router interface {
	// LoadRouterConf loads the routing configuration
	LoadRouterConf(provider string) error

	// InitRouterConfig initializes the routing configuration
	InitRouterConfig(ctx context.Context, rf *entity.ProxyConfig) (err error)

	// GetMatchRouter matches the route
	GetMatchRouter(ctx context.Context) (*entity.TargetService, error)
}

// Option is a routing configuration option
type Option func(o *Options)

// Options is a routing configuration setting
type Options struct {
	// RadixTree is the radix tree for route matching
	RadixTree *radix.Tree
	// RegRouterList is the list of regular routes
	RegRouterList []*RegRouter
	// Clients is the upstream service configuration
	Clients map[string]*entity.BackendConfig
}

// RegRouter represents a regular route, where multiple route items can match the same regular expression
type RegRouter struct {
	// ItemList is the list of route matching items
	ItemList []*entity.RouterItem
	// RegexpStr is the regular expression string
	RegexpStr string
	// Regexp is the compiled regular expression, thread-safe
	*regexp.Regexp
}

// WithRadixTreeRouter configures the radix tree router
func WithRadixTreeRouter(item *entity.RouterItem) Option {
	return func(o *Options) {
		if item == nil {
			return
		}
		if o.RadixTree == nil {
			o.RadixTree = radix.New()
		}
		// Get the leaf node
		leafVal, ok := o.RadixTree.Get(item.Method)
		if ok {
			routerList, aOk := leafVal.([]*entity.RouterItem)
			if !aOk {
				panic("assert radix tree leaf value type err")
			}
			routerList = append(routerList, item)
			o.RadixTree.Insert(item.Method, routerList)
			return
		}
		// If the leaf node does not exist, add it
		o.RadixTree.Insert(item.Method, []*entity.RouterItem{
			item,
		})
	}
}

// WithRegRouter configures the regular expression router
func WithRegRouter(item *entity.RouterItem) Option {
	return func(o *Options) {
		if item == nil {
			return
		}
		if o.RegRouterList == nil {
			o.RegRouterList = []*RegRouter{}
		}
		// Iterate and check if the expression already exists
		for _, regRouter := range o.RegRouterList {
			// If it already exists, append the new route
			if regRouter.RegexpStr == item.Method {
				regRouter.ItemList = append(regRouter.ItemList, item)
				return
			}
		}
		// If it does not exist, add the route
		o.RegRouterList = append(o.RegRouterList, &RegRouter{
			ItemList:  []*entity.RouterItem{item},
			RegexpStr: item.Method,
			Regexp:    regexp.MustCompile(item.Method),
		})
	}
}

// WithRouterClient configures the upstream service
func WithRouterClient(item *entity.BackendConfig) Option {
	return func(o *Options) {
		if item == nil {
			return
		}
		if o.Clients == nil {
			o.Clients = make(map[string]*entity.BackendConfig)
		}
		o.Clients[item.ServiceName] = item
	}
}
