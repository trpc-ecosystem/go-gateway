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
	"context"
	"sync"
)

var (
	loaders   = make(map[string]Loader)
	muxLoader sync.RWMutex
)

// RegisterConfLoader register a router config loader
func RegisterConfLoader(provider string, loader Loader) {
	muxLoader.Lock()
	loaders[provider] = loader
	muxLoader.Unlock()
}

// GetConfLoader returns a router config loader
func GetConfLoader(provider string) Loader {
	muxLoader.RLock()
	l := loaders[provider]
	muxLoader.RUnlock()
	return l
}

// Loader is the interface for configuration loaders.
//
//go:generate mockgen -destination=./configmock/loader_mock.go -package=configmock . Loader
type Loader interface {
	// LoadConf 加载配置
	LoadConf(ctx context.Context, protocol string) error
}
