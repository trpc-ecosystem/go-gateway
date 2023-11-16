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

package protocol

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-gateway/core/service/protocol/mock"
)

func TestGetCliProtocolHandler(t *testing.T) {
	RegisterCliProtocolHandler("trpc", &mock.MockCliProtocolHandler{})
	RegisterCliProtocolHandler("fasthttp", &mock.MockCliProtocolHandler{})

	h, err := GetCliProtocolHandler("trpc")
	assert.Nil(t, err)
	assert.NotNil(t, h)

	h, err = GetCliProtocolHandler("fasthttp")
	assert.Nil(t, err)
	assert.NotNil(t, h)
}

func TestDefaultProtocolHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCliProtocol := mock.NewMockCliProtocolHandler(ctrl)
	RegisterCliProtocolHandler("fasthttp", mockCliProtocol)
	h, err := GetCliProtocolHandler("fasthttp")
	assert.Nil(t, err)
	assert.NotNil(t, h)
	h, err = GetCliProtocolHandler("invalid")
	assert.NotNil(t, err)
}
