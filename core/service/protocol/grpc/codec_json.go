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

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	json "github.com/json-iterator/go"
	"github.com/oxtoacart/bpool"
	"google.golang.org/grpc/encoding"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/log"
)

func init() {
	encoding.RegisterCodec(jsonCodec{})
}

// Create a buffer Pool with 16 instances, each preallocated with 256 bytes
var bufferPool = bpool.NewSizedBufferPool(16, 256)

var jsonpbMarshaler = &jsonpb.Marshaler{}

type jsonCodec struct{}

// Marshal serializes the data
func (jsonCodec) Marshal(v interface{}) ([]byte, error) {
	if pb, ok := v.(proto.Message); ok {
		buf := bufferPool.Get()
		defer bufferPool.Put(buf)
		if err := jsonpbMarshaler.Marshal(buf, pb); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
	bt, ok := v.([]byte)
	if ok {
		log.Debugf("byte req body:%s", string(bt))
		return bt, nil
	}
	return json.Marshal(v)
}

// Unmarshal deserializes the data
func (jsonCodec) Unmarshal(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	ctx, ok := v.(context.Context)
	if !ok {
		return errs.New(gerrs.ErrGatewayUnknown, "invalid context")
	}
	header := Head(ctx)
	if header == nil {
		return errs.New(gerrs.ErrGatewayUnknown, "get no grpc header unmarshal")
	}
	header.Rsp = data
	return nil
}

// Name returns the codec name
func (jsonCodec) Name() string {
	return "json"
}
