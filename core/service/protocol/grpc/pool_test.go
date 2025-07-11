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

package grpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_pool_Get(t *testing.T) {

	p := &Pool{}
	got, err := p.Get("localhost:0", 2*time.Second)
	assert.Nil(t, err)
	assert.NotNil(t, got)

	got, err = p.Get("localhost:0", 2*time.Second)
	assert.Nil(t, err)
	assert.NotNil(t, got)

	_, err = p.Get("127.0.0.1:8080", 10)
	assert.NotNil(t, err)
}
