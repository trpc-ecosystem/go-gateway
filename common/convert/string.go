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

// Package convert some parameter conversion utility functions.
package convert

import (
	"encoding/json"
	"hash/fnv"
	"strconv"
)

// ToJSONStr converts to JSON string for logging purposes.
func ToJSONStr(o interface{}) string {
	// 用于日志打印，忽略错误
	b, _ := json.Marshal(o)
	return string(b)
}

// ToIntSlice converts to a list of integers.
func ToIntSlice(list []string) ([]int, error) {
	var idxList []int
	for _, i := range list {
		idx, err := strconv.Atoi(i)
		if err != nil {
			return nil, err
		}
		idxList = append(idxList, idx)
	}
	return idxList, nil
}

// Fnv32 Fnv-hash algorithm
// FNV can quickly hash large amounts of data while maintaining a low collision rate.
// Its high dispersion makes it suitable for hashing very similar strings.
func Fnv32(str string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(str))
	return h.Sum32()
}
