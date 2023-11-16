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

package trpc

import (
	trpc "trpc.group/trpc-go/trpc-go"
)

// IsProduction determines whether it is a production environment
type IsProduction func() bool

// DefaultIsProduction determines whether it is a production environment and can be overridden
var DefaultIsProduction IsProduction = func() bool {
	return trpc.GlobalConfig().Global.Namespace == "Production"
}
