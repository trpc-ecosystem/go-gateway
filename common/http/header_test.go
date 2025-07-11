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

package http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeHTTPHeaders(t *testing.T) {
	type args struct {
		h http.Header
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{
			name: "header exist",
			args: args{
				h: http.Header{
					"Content-Type": []string{
						"application/json",
					},
				},
			},
			want: "Content-Type=application%2Fjson",
		},
		{
			name: "header not exist",
			args: args{},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, EncodeHTTPHeaders(tt.args.h), "EncodeHTTPHeaders(%v)", tt.args.h)
		})
	}
}
