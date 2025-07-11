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

package config

import (
	"sync"
	"time"

	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-go/transport"
)

const (
	defaultMaxRequestBodySize = 1 << 27
	defaultReadBufferSize     = 1 << 20
)

// CustomTransportOpts defines a function type for custom options
type CustomTransportOpts func(opts ...ServerOption)

// LoadRouterFunc is a function type for dynamically loading routers
type LoadRouterFunc func(provider string) error

var (
	svrTransOpts    = make(map[string]CustomTransportOpts)
	routeLoaders    = make(map[string]LoadRouterFunc)
	muxSvrTransOpts = sync.RWMutex{}
	muxRouteLoader  = sync.RWMutex{}
)

// RegisterCustomTransOpts registers custom protocol options
func RegisterCustomTransOpts(protocol string, f CustomTransportOpts) {
	muxSvrTransOpts.Lock()
	svrTransOpts[protocol] = f
	muxSvrTransOpts.Unlock()
}

// GetCustomTransOpts returns custom protocol options
func GetCustomTransOpts(protocol string) CustomTransportOpts {
	muxSvrTransOpts.RLock()
	f := svrTransOpts[protocol]
	muxSvrTransOpts.RUnlock()
	return f
}

// RegisterRouteLoader registers a route loading function
func RegisterRouteLoader(protocol string, f LoadRouterFunc) {
	muxRouteLoader.Lock()
	routeLoaders[protocol] = f
	muxRouteLoader.Unlock()
}

// GetRouteLoader returns a route loading function
func GetRouteLoader(protocol string) LoadRouterFunc {
	muxRouteLoader.RLock()
	f := routeLoaders[protocol]
	muxRouteLoader.RUnlock()
	return f
}

// ServerOptions are server settings options.
type ServerOptions struct {
	transport.ServerTransportOptions
	// Handler is the handler to use for the server.
	Handler fasthttp.RequestHandler
	// MaxCons is the maximum number of concurrent connections.
	MaxCons int
	// MaxConsPerIP is the maximum number of concurrent connections per IP.
	MaxConsPerIP int
	// ReadTimeout is the maximum duration for reading the request.
	ReadTimeout time.Duration
	// WriteTimeout is the maximum duration for writing the response.
	WriteTimeout time.Duration
	// MaxRequestBodySize is the maximum size of the request body.
	MaxRequestBodySize int
	// ReadBufferSize is the size of the buffer for reading the request.
	ReadBufferSize int
}

// ServerOption defines a function that can be used to configure a ServerOptions.
type ServerOption func(o *ServerOptions)

// WithReusePort sets the reuse port option.
func WithReusePort(b bool) ServerOption {
	return func(o *ServerOptions) {
		o.ReusePort = b
	}
}

// WithMaxCons sets	the maximum number of concurrent connections.
func WithMaxCons(cons int) ServerOption {
	return func(o *ServerOptions) {
		o.MaxCons = cons
	}
}

// WithMaxConsPerIP sets the maximum number of concurrent connections per IP.
func WithMaxConsPerIP(cons int) ServerOption {
	return func(o *ServerOptions) {
		o.MaxConsPerIP = cons
	}
}

// WithReadTimeout sets the maximum duration for reading the request.
func WithReadTimeout(timeout time.Duration) ServerOption {
	return func(o *ServerOptions) {
		o.ReadTimeout = timeout
	}
}

// WithWriteTimeout sets the maximum duration for writing the response.
func WithWriteTimeout(timeout time.Duration) ServerOption {
	return func(o *ServerOptions) {
		o.WriteTimeout = timeout
	}
}

// WithMaxRequestBodySize sets the maximum size of the request body.
func WithMaxRequestBodySize(size int) ServerOption {
	return func(o *ServerOptions) {
		if size == 0 {
			size = defaultMaxRequestBodySize
		}
		o.MaxRequestBodySize = size
	}
}

// WithReadBufferSize sets the size of the buffer for reading the request.
func WithReadBufferSize(size int) ServerOption {
	return func(o *ServerOptions) {
		if size == 0 {
			size = defaultReadBufferSize
		}
		o.ReadBufferSize = size
	}
}
