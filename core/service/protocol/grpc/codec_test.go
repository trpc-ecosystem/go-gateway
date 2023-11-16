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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerCodec_Decode(t *testing.T) {
	s := &ServerCodec{}
	gotReqbody, err := s.Decode(nil, []byte("xx"))
	assert.Nil(t, err)
	assert.Equal(t, gotReqbody, []byte("xx"))

	gotReqbody, err = s.Encode(nil, []byte("xx"))
	assert.Nil(t, err)
	assert.Equal(t, gotReqbody, []byte("xx"))
}

func TestClientCodec_Encode(t *testing.T) {
	s := &ClientCodec{}
	gotReqbody, err := s.Decode(nil, []byte("xx"))
	assert.Nil(t, err)
	assert.Equal(t, gotReqbody, []byte("xx"))

	gotReqbody, err = s.Encode(nil, []byte("xx"))
	assert.Nil(t, err)
	assert.Equal(t, gotReqbody, []byte("xx"))
}
