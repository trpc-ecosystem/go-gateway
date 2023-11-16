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
	"errors"
	"sync"

	"github.com/valyala/fasthttp"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-go/errs"
)

// DefaultConnPoolSize is the default connection pool size. Note that this value does not limit the maximum number of
// connections.
var DefaultConnPoolSize = 1000

// ErrClosed represents a closed connection error.
var ErrClosed = errs.New(gerrs.ErrConnClosed, "connection closed")

//go:generate mockgen -destination=./mock/pool_mock.go -package=mock . Pool

// Pool defines the connection pool interface.
type Pool interface {
	// Get returns a proxy client.
	Get(addr string) (*fasthttp.HostClient, error)
	// Put puts the proxy client back into the pool.
	Put(proxy *fasthttp.HostClient) error
	Close()
	Len() int
}

// ProxyChan represents a channel for proxy clients.
type ProxyChan chan *fasthttp.HostClient

// ConnPool is a connection pool implementation based on channels.
type ConnPool struct {
	sync.RWMutex

	poolSize     int
	proxyChanMap map[string]ProxyChan
}

// NewConnPool creates a new connection pool.
var NewConnPool = func(maxCap int) Pool {
	if maxCap == 0 {
		maxCap = DefaultConnPoolSize
	}

	pool := &ConnPool{
		poolSize:     maxCap,
		proxyChanMap: make(map[string]ProxyChan),
	}

	return pool
}

// Get gets a proxy client.
func (p *ConnPool) Get(addr string) (*fasthttp.HostClient, error) {
	if p.proxyChanMap == nil {
		return nil, ErrClosed
	}

	p.Lock()
	defer p.Unlock()
	proxyChan, ok := p.proxyChanMap[addr]
	if !ok {
		p.proxyChanMap[addr] = make(chan *fasthttp.HostClient, p.poolSize)
	}

	select {
	case proxy := <-proxyChan:
		if proxy == nil {
			return nil, ErrClosed
		}
		return proxy, nil
	default:
		// If no available proxy, create a new one
		proxy := &fasthttp.HostClient{
			Addr: addr,
			// Proxy only makes one request and does not allow retries
			MaxIdemponentCallAttempts: 1,
		}
		return proxy, nil
	}
}

// Put puts the proxy client back into the connection pool.
func (p *ConnPool) Put(proxy *fasthttp.HostClient) error {
	if proxy == nil {
		return errors.New("nil proxy client")
	}

	p.Lock()
	defer p.Unlock()
	_, ok := p.proxyChanMap[proxy.Addr]
	if !ok {
		p.proxyChanMap[proxy.Addr] = make(chan *fasthttp.HostClient, p.poolSize)
	}
	if p.proxyChanMap[proxy.Addr] == nil {
		p.Close()
		return nil
	}

	select {
	case p.proxyChanMap[proxy.Addr] <- proxy:
		return nil
	default:
		// If the connection pool is full, close existing connections
		p.Close()
		return nil
	}
}

// Close closes the connection pool.
func (p *ConnPool) Close() {
	for _, pChan := range p.proxyChanMap {
		if pChan == nil {
			return
		}
		close(pChan)
		for range pChan {
		}
	}
	p.proxyChanMap = make(map[string]ProxyChan)
}

// Len returns the current number of connections in the connection pool.
func (p *ConnPool) Len() int {
	num := 0
	for _, ch := range p.proxyChanMap {
		num += len(ch)
	}
	return num
}
