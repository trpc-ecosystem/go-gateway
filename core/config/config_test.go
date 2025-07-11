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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
)

func fakeTransOpts(...ServerOption) {}

type fakeRouter struct{}

// LoadRouterConf is a fake router loader for unit test
func (r *fakeRouter) LoadRouterConf(string) error {
	return nil
}

// GetMatchRouter is a fake router matcher for unit test
func (r *fakeRouter) GetMatchRouter(_ context.Context) (*client.BackendConfig, error) {
	return &client.BackendConfig{}, nil
}

const configPath = "../../testdata/trpc_go.yaml"
const configPathInvalid = "../../testdata/trpc_go_error.yaml"

func TestSetup(t *testing.T) {
	RegisterCustomTransOpts("fasthttp", fakeTransOpts)

	// register router loader success
	RegisterRouteLoader("fasthttp", func(provider string) error {
		return nil
	})
	trpc.ServerConfigPath = configPathInvalid
	err := setup()
	assert.NotNil(t, err)

	trpc.ServerConfigPath = configPath
	err = setup()
	assert.Nil(t, err)

	//	load router failed
	RegisterRouteLoader("fasthttp", func(provider string) error {
		return errors.New("not found")
	})
	err = setup()
	assert.NotNil(t, err)

	// empty router loader
	RegisterRouteLoader("fasthttp", nil)
	err = setup()
	assert.NotNil(t, err)

	// load router config failed
	trpc.ServerConfigPath = ""
	err = setup()
	assert.NotNil(t, err)
}

func TestNewServer(t *testing.T) {
	RegisterCustomTransOpts("fasthttp", fakeTransOpts)
	trpc.ServerConfigPath = configPath
	r := &fakeRouter{}
	RegisterRouteLoader("fasthttp", r.LoadRouterConf)

	// start server success
	s := NewServer()
	assert.NotNil(t, s)

	func() {
		defer func() {
			if re := recover(); re != nil {
				assert.NotNil(t, re)
			}
		}()
		// 配置文件获取错误
		trpc.ServerConfigPath = ""
		s = NewServer()
	}()

	func() {
		defer func() {
			if re := recover(); re != nil {
				t.Log(re)
				assert.NotNil(t, re)
			}
		}()
		trpc.ServerConfigPath = configPathInvalid
		s = NewServer()
	}()
}
