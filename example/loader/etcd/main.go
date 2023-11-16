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

// Package etcd is gateway usage example
package main

import (
	"flag"

	"trpc.group/trpc-go/trpc-gateway/core/config"
	// Register loader
	_ "trpc.group/trpc-go/trpc-gateway/core/loader/etcd"
	"trpc.group/trpc-go/trpc-gateway/core/service/fhttp"
	"trpc.group/trpc-go/trpc-go/log"
	// Register upstream protocol
	_ "trpc.group/trpc-go/trpc-gateway/core/service/protocol/fasthttp"
	_ "trpc.group/trpc-go/trpc-gateway/core/service/protocol/trpc"
	_ "trpc.group/trpc-go/trpc-gateway/plugin/demo"
)

func main() {
	flag.Parse()
	s := config.NewServer()
	// register gateway service
	fhttp.RegisterFastHTTPService(s)
	if err := s.Serve(); err != nil {
		log.Fatal(err)
	}
}
