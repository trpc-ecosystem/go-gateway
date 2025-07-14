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

package trpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-go/codec"
)

func TestGetSerializationType(t *testing.T) {
	got, err := GetSerializationType("application/json")
	assert.Nil(t, err)
	assert.Equal(t, codec.SerializationTypeJSON, got)
	got, err = GetSerializationType("application/json; charset=UTF-8")
	assert.Nil(t, err)
	assert.Equal(t, codec.SerializationTypeJSON, got)

	err = Register("application/json", 111)
	assert.Nil(t, err)
	got, err = GetSerializationType("application/json; charset=UTF-8")
	assert.Nil(t, err)
	assert.Equal(t, 111, got)

	got, err = GetSerializationType("text/plain; charset=UTF-8")
	assert.NotNil(t, err)
}
