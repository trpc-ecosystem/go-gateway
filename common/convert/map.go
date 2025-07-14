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

package convert

// StrSlice2Map converts a string slice into a map. Note: it will filter out empty strings.
func StrSlice2Map(list []string) map[string]struct{} {
	m := make(map[string]struct{}, len(list))
	for _, s := range list {
		if s == "" {
			continue
		}
		m[s] = struct{}{}
	}
	return m
}
