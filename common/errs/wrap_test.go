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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-go/errs"
)

func TestWrap(t *testing.T) {
	err := gerrs.Wrap(errors.New("err"), "my err")
	assert.NotNil(t, err)

	err = gerrs.Wrap(nil, "my err")
	assert.NotNil(t, err)

	var e *errs.Error
	err = gerrs.Wrap(e, "my err")
	assert.NotNil(t, err)
}

func TestWrapf(t *testing.T) {
	err := gerrs.Wrapf(errors.New("err"), "my err:%s", "args")
	assert.NotNil(t, err)
}

func TestUnWrap(t *testing.T) {
	_, ok := gerrs.UnWrap(nil)
	assert.False(t, ok)
	_, ok = gerrs.UnWrap(errors.New("err"))
	assert.False(t, ok)
	terr, ok := gerrs.UnWrap(errs.New(1, "trpc err"))
	assert.True(t, ok)
	assert.Equal(t, 1, int(terr.Code))
}
