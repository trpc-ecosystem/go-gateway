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

package fhttp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestNewConnPool(t *testing.T) {
	p := NewConnPool(0)
	assert.NotNil(t, p)

	p = NewConnPool(200)
	assert.NotNil(t, p)
}

func TestConnPool_Get(t *testing.T) {
	p := NewConnPool(200)
	proxy, err := p.Get("localhost:80")
	assert.Nil(t, err)
	assert.NotNil(t, proxy)

	p = &ConnPool{}
	proxy, err = p.Get("localhost")
	assert.Nil(t, proxy)
	assert.NotNil(t, err)
}

func TestConnPool_Put(t *testing.T) {
	p := NewConnPool(3).(*ConnPool)
	proxy := &fasthttp.HostClient{}
	err := p.Put(proxy)
	assert.Nil(t, err)
	// Put empty proxy
	err = p.Put(nil)
	assert.NotNil(t, err)
	// Current ip:port exists, but the connection pool is empty
	proxy.Addr = "ip:port"
	p.proxyChanMap = map[string]ProxyChan{
		"ip:port": nil,
	}
	err = p.Put(proxy)
	assert.Nil(t, err)
}

func TestConnPool_Len(t *testing.T) {
	p := NewConnPool(200)
	num := p.Len()
	assert.Equal(t, 0, num)
}

func TestConnPool_Close(t *testing.T) {
	p := NewConnPool(200).(*ConnPool)
	p.proxyChanMap = make(map[string]ProxyChan)
	p.proxyChanMap["ip:port1"] = nil
	pool := make(chan *fasthttp.HostClient, 1)
	pool <- &fasthttp.HostClient{}
	p.proxyChanMap["ip:port2"] = pool
	p.Close()
}
