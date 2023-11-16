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

package grpc

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	_ "github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	mockproto "trpc.group/trpc-go/trpc-gateway/core/service/protocol/grpc/mock"
)

//go:generate mockgen -destination=./mock/messge.go -package=mock_grpc  github.com/golang/protobuf/proto Message
func Test_jsonCodec_Marshal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockMessage := mockproto.NewMockMessage(ctrl)
	js := jsonCodec{}
	got, err := js.Marshal(mockMessage)
	assert.Nil(t, err)
	assert.NotNil(t, got)

	got, err = js.Marshal([]byte("xxx"))
	assert.Nil(t, err)
	assert.NotNil(t, got)

	got, err = js.Marshal(map[string]string{"a": "b"})
	assert.Nil(t, err)
	assert.NotNil(t, got)
}

func Test_jsonCodec_Unmarshal(t *testing.T) {

	js := jsonCodec{}
	err := js.Unmarshal([]byte(""), nil)
	assert.Nil(t, err)

	err = js.Unmarshal([]byte("xx"), "aa")
	assert.NotNil(t, err)

	err = js.Unmarshal([]byte("xx"), context.Background())
	assert.NotNil(t, err)

	ctx := WithHeader(context.Background(), &Header{
		Req:         nil,
		Rsp:         nil,
		InMetadata:  nil,
		OutMetadata: nil,
	})
	err = js.Unmarshal([]byte("xxx"), ctx)
	assert.Nil(t, err)
}
