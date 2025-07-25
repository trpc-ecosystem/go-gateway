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

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDeduplicate
func TestDeduplicate(t *testing.T) {
	a := []string{"a1", "a2"}
	b := []string{"b1", "b2", "a2"}
	r := Deduplicate(a, b)
	assert.Equal(t, r, []string{"a1", "a2", "b1", "b2"})
}
