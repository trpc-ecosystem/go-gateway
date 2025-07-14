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
	"testing"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-gateway/core/entity"
)

func TestWithRadixTreeRouter(t *testing.T) {
	o := &Options{}
	// item is nil
	WithRadixTreeRouter(nil)(o)

	routerItem := &entity.RouterItem{
		Method: "/user/info",
	}
	// radixTree is nil
	WithRadixTreeRouter(routerItem)(o)
	leaf, ok := o.RadixTree.Get("/user/info")
	assert.True(t, ok)
	routerList, ok := leaf.([]*entity.RouterItem)
	assert.True(t, ok)
	assert.Equal(t, len(routerList), 1)

	// Set successfully
	routerItem2 := &entity.RouterItem{
		Method: "/user/info",
		Host:   []string{"r.inews.qq.com"},
	}
	// Update node
	WithRadixTreeRouter(routerItem2)(o)
	leaf, ok = o.RadixTree.Get("/user/info")
	assert.True(t, ok)
	routerList, ok = leaf.([]*entity.RouterItem)
	assert.True(t, ok)
	assert.Equal(t, len(routerList), 2)
}

func TestWithRegRouter(t *testing.T) {
	o := &Options{}
	// item is nil
	WithRegRouter(nil)(o)

	routerItem := &entity.RouterItem{
		Method:   "/user/info",
		IsRegexp: true,
	}
	// RegRouterList is nil
	WithRegRouter(routerItem)(o)
	assert.Equal(t, len(o.RegRouterList), 1)

	// Set successfully
	routerItem2 := &entity.RouterItem{
		Method:   "/user/info",
		Host:     []string{"r.inews.qq.com"},
		IsRegexp: true,
	}
	// Update node
	WithRegRouter(routerItem2)(o)
	assert.Equal(t, len(o.RegRouterList[0].ItemList), 2)
}

func TestWithRouterClient(t *testing.T) {
	o := &Options{}
	// item is nil
	WithRouterClient(nil)(o)
	// Set successfully
	c := &entity.BackendConfig{}
	WithRouterClient(c)(o)
	assert.Equal(t, len(o.Clients), 1)

	r := GetRouter("fasthttp")
	assert.NotNil(t, r)
}
