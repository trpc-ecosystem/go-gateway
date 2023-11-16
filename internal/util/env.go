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

package util

import "os"

// ExpandEnv looks for ${var} in s and replaces them with value of the
// corresponding environment variable.
// $var is considered invalid.
// It's not like os.ExpandEnv which will handle both ${var} and $var.
// Since configurations like password for redis/mysql may contain $, this
// method is needed.
// TODO test case
func ExpandEnv(s string) string {
	var buf []byte
	i := 0
	for j := 0; j < len(s); j++ {
		if s[j] == '$' && j+2 < len(s) && s[j+1] == '{' { // only ${var} instead of $var is valid
			if buf == nil {
				buf = make([]byte, 0, 2*len(s))
			}
			buf = append(buf, s[i:j]...)
			name, w := getEnvName(s[j+1:])
			if name == "" && w > 0 {
				// invalid matching, remove the $
			} else if name == "" {
				buf = append(buf, s[j]) // keep the $
			} else {
				buf = append(buf, os.Getenv(name)...)
			}
			j += w
			i = j + 1
		}
	}
	if buf == nil {
		return s
	}
	return string(buf) + s[i:]
}

// getEnvName gets env name, that is, var from ${var}.
// And content of var and its len will be returned.
func getEnvName(s string) (string, int) {
	// look for right curly bracket '}'
	// it's guaranteed that the first char is '{' and the string has at least two char
	for i := 1; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\n' || s[i] == '"' { // "xx${xxx"
			return "", 0 // encounter invalid char, keep the $
		}
		if s[i] == '}' {
			if i == 1 { // ${}
				return "", 2 // remove ${}
			}
			return s[1:i], i + 1
		}
	}
	return "", 0 // no }，keep the $
}
