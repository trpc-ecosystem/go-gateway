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

// example of gateway
package main

import (
	"context"

	"trpc.group/trpc-go/trpc-go/errs"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

var greeter = &greeterServiceImpl{
	proxy: pb.NewGreeterClientProxy(),
}

// greeterServiceImpl greeter service implement
type greeterServiceImpl struct {
	proxy pb.GreeterClientProxy
}

// SayHello 响应成功的示例
func (s *greeterServiceImpl) SayHello(_ context.Context, _ *pb.HelloRequest) (*pb.HelloReply, error) {
	rsp := &pb.HelloReply{
		Msg: "Hello",
	}
	return rsp, nil
}

// SayHi 返回 err 的示例
func (s *greeterServiceImpl) SayHi(_ context.Context, _ *pb.HelloRequest) (*pb.HelloReply, error) {
	return nil, errs.New(8888, "say hi err")
}
