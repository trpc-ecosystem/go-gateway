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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockLoader struct {
}

func (m mockLoader) LoadConf(context.Context, string) error {
	return nil
}

func TestRegisterConfLoader(t *testing.T) {
	ml := mockLoader{}
	RegisterConfLoader("test", ml)
	l := GetConfLoader("test")
	assert.Equal(t, l, ml)
}
