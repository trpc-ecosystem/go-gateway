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

// Deduplicate merges two slices.
// Order will be kept and duplication will be removed.
func Deduplicate(a, b []string) []string {
	r := make([]string, 0, len(a)+len(b))
	m := make(map[string]bool)
	for _, s := range append(a, b...) {
		if _, ok := m[s]; !ok {
			m[s] = true
			r = append(r, s)
		}
	}
	return r
}
