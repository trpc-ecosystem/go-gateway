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

package convert_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-gateway/common/convert"
)

func TestToJSONStr(t *testing.T) {
	assert.Equal(t, convert.ToJSONStr(1), "1")
}

func TestToIntSlice(t *testing.T) {
	s := []string{"1", "2"}
	got, err := convert.ToIntSlice(s)
	assert.Nil(t, err)
	assert.ElementsMatch(t, got, []int{1, 2})

	s = []string{"1", "2", "c"}
	_, err = convert.ToIntSlice(s)
	assert.NotNil(t, err)
}

func TestFnv32(t *testing.T) {
	ret := convert.Fnv32("str")
	t.Log(ret)
	assert.Equal(t, uint32(3259748752), ret)
}
