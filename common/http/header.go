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
	"net/url"
)

const (
	// TRPCGatewayHTTPHeader http header
	TRPCGatewayHTTPHeader = "TRPC_GATEWAY_HTTP_HEADER"
	// TRPCGatewayHTTPQuery http raw uri
	TRPCGatewayHTTPQuery = "TRPC_GATEWAY_HTTP_QUERY"
)

// EncodeHTTPHeaders encode http headers to string
func EncodeHTTPHeaders(h http.Header) string {
	if len(h) == 0 {
		return ""
	}
	u := make(url.Values)
	for k, v := range h {
		u[k] = v
	}
	return u.Encode()
}
