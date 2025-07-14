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

package objectpool_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-gateway/internal/pool/objectpool"
)

func TestBufferPool_Get(t *testing.T) {
	p := objectpool.NewBufferPool()

	buf := p.Get()
	assert.NotNil(t, buf)
	buf.Reset()
	p.Put(buf)
}
