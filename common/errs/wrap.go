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

package errs

import (
	"errors"
	"fmt"

	"trpc.group/trpc-go/trpc-go/errs"
)

// Wrap the error to use a unified log separator,
// refer to this article for the usage of the errors standard library:
// https://www.flysnow.org/2019/09/06/go1.13-error-wrapping.html
func Wrap(e error, msg string) error {
	return fmt.Errorf("%s||%w", msg, wrapAsTRPCErr(e))
}

// Wrapf  Wrap the error to use a unified log separator, with the ability to pass parameters
func Wrapf(err error, format string, arg ...interface{}) error {
	return fmt.Errorf("%s||%w", fmt.Sprintf(format, arg...), wrapAsTRPCErr(err))
}

// UnWrap Obtain the most original trpc error
func UnWrap(e error) (*errs.Error, bool) {
	if e == nil {
		return nil, false
	}
	var terr *errs.Error
	if !errors.As(e, &terr) {
		return nil, false
	}
	return terr, true
}

// If it is not a trpc error, wrap it as a trpc error
func wrapAsTRPCErr(e error) error {
	if e == nil {
		return nil
	}
	var terr *errs.Error
	if !errors.As(e, &terr) {
		return errs.Wrap(e, ErrGatewayUnknown, e.Error())
	}
	return e
}
