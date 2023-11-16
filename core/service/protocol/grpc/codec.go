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
	"trpc.group/trpc-go/trpc-go/codec"
)

var (
	// DefaultServerCodec is the default encoding/decoding instance for the server
	DefaultServerCodec = &ServerCodec{}
	// DefaultClientCodec is the default encoding/decoding instance for the client
	DefaultClientCodec = &ClientCodec{}
)

// init registers the grpc codec
func init() {
	codec.Register(Protocol, DefaultServerCodec, DefaultClientCodec)
}

// ServerCodec is the server-side codec for encoding/decoding
type ServerCodec struct{}

// Decode is used to decode the message
func (s *ServerCodec) Decode(_ codec.Msg, reqbuf []byte) ([]byte, error) {
	return reqbuf, nil
}

// Encode is used to encode the message
func (s *ServerCodec) Encode(_ codec.Msg, reqbuf []byte) ([]byte, error) {
	return reqbuf, nil
}

// ClientCodec is the codec for the grpc client, it does nothing
type ClientCodec struct{}

// Encode is the encoder for the grpc client, it does nothing
func (c *ClientCodec) Encode(_ codec.Msg, reqbody []byte) ([]byte, error) {
	return reqbody, nil
}

// Decode is the decoder for the grpc client, it does nothing
func (c *ClientCodec) Decode(_ codec.Msg, rspbody []byte) ([]byte, error) {
	return rspbody, nil
}
