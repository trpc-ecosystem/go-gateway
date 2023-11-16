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

package errs_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-go/errs"
)

func TestRegister(t *testing.T) {
	// Error code already exists
	err := gerrs.Register(errs.RetServerDecodeFail, 1)
	assert.NotNil(t, err)
	// Registration successful
	err = gerrs.Register(4001, 1)
	assert.Nil(t, err)
}

func TestGetCodeMap(t *testing.T) {
	// Successful retrieval
	httpCode := gerrs.GetHTTPStatus(errs.RetServerDecodeFail)
	assert.Equal(t, fasthttp.StatusBadRequest, httpCode)
	// fallback obtained
	httpCode = gerrs.GetHTTPStatus(4002)
	assert.Equal(t, fasthttp.StatusInternalServerError, httpCode)
}

func TestRegisterHTTPSuccessStatus(t *testing.T) {
	gerrs.RegisterSuccessHTTPStatus([]int32{302})
	gerrs.IsSuccessHTTPStatus(302)
}
