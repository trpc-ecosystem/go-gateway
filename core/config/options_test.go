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

package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	opts := []ServerOption{
		WithReusePort(true),
		WithMaxCons(10),
		WithMaxConsPerIP(5),
		WithMaxRequestBodySize(0),
		WithReadBufferSize(0),
		WithReadTimeout(time.Second),
		WithWriteTimeout(time.Second),
	}

	opt := &ServerOptions{}
	for _, o := range opts {
		o(opt)
	}

	assert.Equal(t, 10, opt.MaxCons)
}

func fakeTransOpt(...ServerOption) {
}

func TestRegisterCustomTransOpts(t *testing.T) {
	RegisterCustomTransOpts("test", fakeTransOpt)
	f := GetCustomTransOpts("test")
	assert.NotNil(t, f)
}
