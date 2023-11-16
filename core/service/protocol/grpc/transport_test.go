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

package grpc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prashantv/gostub"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"trpc.group/trpc-go/trpc-gateway/core/service/protocol/grpc"
	mock_grpc "trpc.group/trpc-go/trpc-gateway/core/service/protocol/grpc/mock"
	"trpc.group/trpc-go/trpc-go/transport"
)

//go:generate mockgen -destination=./mock/grpcclient.go -package=mock_grpc  google.golang.org/grpc ClientConnInterface
func Test_clientTransport_RoundTrip(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	stub := gostub.New()
	defer stub.Reset()

	c := grpc.DefaultClientTransport
	opts := []transport.RoundTripOption{
		transport.WithDialAddress("localhost:0"),
		transport.WithDialTimeout(0),
	}

	_, err := c.RoundTrip(context.Background(), nil, opts...)
	assert.NotNil(t, err)

	ctx := grpc.WithHeader(context.Background(), &grpc.Header{
		Req:        nil,
		Rsp:        nil,
		InMetadata: nil,
		OutMetadata: metadata.MD{
			"a": []string{"b"},
		},
	})
	_, err = c.RoundTrip(ctx, nil, opts...)
	assert.NotNil(t, err)

	mockPool := mock_grpc.NewMockConnPool(ctrl)
	mockPool.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, errors.New("err"))
	c.ConnectionPool = mockPool
	_, err = c.RoundTrip(ctx, nil, opts...)
	assert.NotNil(t, err)

	mockClient := mock_grpc.NewMockClientConnInterface(ctrl)
	mockClient.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	mockPool.EXPECT().Get(gomock.Any(), gomock.Any()).Return(mockClient, nil)
	c.ConnectionPool = mockPool
	_, err = c.RoundTrip(ctx, nil, opts...)
	assert.Nil(t, err)

	mockClient.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any()).Return(status.Error(codes.DeadlineExceeded, "timeout"))
	mockPool.EXPECT().Get(gomock.Any(), gomock.Any()).Return(mockClient, nil)
	c.ConnectionPool = mockPool
	_, err = c.RoundTrip(ctx, nil, opts...)
	assert.NotNil(t, err)

	mockClient.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any()).Return(status.Error(codes.Canceled, "cancel"))
	mockPool.EXPECT().Get(gomock.Any(), gomock.Any()).Return(mockClient, nil)
	c.ConnectionPool = mockPool
	_, err = c.RoundTrip(ctx, nil, opts...)
	assert.NotNil(t, err)
}
