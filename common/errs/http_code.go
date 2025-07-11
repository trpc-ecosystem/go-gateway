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

package errs

import (
	"errors"

	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc/trpc-protocol/pb/go/trpc"
)

// httpStatusMap Mapping tRPC error codes to HTTP status codes, the gateway plugin can be expanded
var httpStatusMap = map[trpc.TrpcRetCode]int{
	errs.RetServerDecodeFail:   fasthttp.StatusBadRequest,
	errs.RetServerEncodeFail:   fasthttp.StatusInternalServerError,
	errs.RetServerNoService:    fasthttp.StatusNotFound,
	errs.RetServerNoFunc:       fasthttp.StatusNotFound,
	errs.RetServerTimeout:      fasthttp.StatusGatewayTimeout,
	errs.RetServerOverload:     fasthttp.StatusTooManyRequests,
	errs.RetServerThrottled:    fasthttp.StatusTooManyRequests,
	errs.RetServerSystemErr:    fasthttp.StatusInternalServerError,
	errs.RetServerAuthFail:     fasthttp.StatusUnauthorized,
	errs.RetServerValidateFail: fasthttp.StatusBadRequest,
	errs.RetClientTimeout:      fasthttp.StatusRequestTimeout,
	errs.RetClientCanceled:     fasthttp.StatusRequestTimeout,
	errs.RetClientNetErr:       fasthttp.StatusInternalServerError,
	errs.RetUnknown:            fasthttp.StatusInternalServerError,
	ErrInvalidReq:              fasthttp.StatusForbidden,
}

// Register Registering the mapping relationship between custom err codes and HTTP status codes,
// for example: in the authentication plugin, mapping a custom authentication failure err code
// to the HTTP 401 status code.
// Call it in the plugin's Setup method.
func Register[T trpc.TrpcRetCode](errCode T, httpStatus int32) error {
	if _, ok := httpStatusMap[trpc.TrpcRetCode(errCode)]; ok {
		return errors.New("err code exist")
	}
	// trpc-go service startup plugins are not loaded concurrently, so there is no need for locking
	httpStatusMap[trpc.TrpcRetCode(errCode)] = int(httpStatus)
	return nil
}

// GetHTTPStatus Obtaining the HTTP status code through tRPC error code
func GetHTTPStatus(trpcCode trpc.TrpcRetCode) int {
	httpCode, ok := httpStatusMap[trpcCode]
	if ok {
		return httpCode
	}
	return fasthttp.StatusInternalServerError
}

// successHTTPStatusMap Defining HTTP status codes as successful redirects, such as 3xx.
// These status codes indicate successful redirection and will not report errors.
var successHTTPStatusMap = map[int32]bool{
	fasthttp.StatusOK: true,
}

// RegisterSuccessHTTPStatus Registering successful HTTP status codes for redirection
func RegisterSuccessHTTPStatus(httpStatusList []int32) {
	for _, status := range httpStatusList {
		successHTTPStatusMap[status] = true
	}
}

// IsSuccessHTTPStatus Determining if an HTTP status code indicates successful redirection
func IsSuccessHTTPStatus(httpStatus int32) bool {
	return successHTTPStatusMap[httpStatus]
}
