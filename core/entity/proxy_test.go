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

package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-gateway/core/entity"
	"trpc.group/trpc-go/trpc-go/client"
)

func TestUnmarshalYAML(t *testing.T) {
	bc := &entity.BackendConfig{
		BackendConfig: client.BackendConfig{},
		Plugins:       nil,
	}
	bcByte, err := yaml.Marshal(bc)
	assert.Nil(t, err)
	err = yaml.Unmarshal(bcByte, &bc)
	assert.Nil(t, err)
}
