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

package file

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-gateway/core/router"
	mock_router "trpc.group/trpc-go/trpc-gateway/core/router/mock"
)

const configPathInvalid = "../../../testdata/trpc_go_error.yaml"
const configPathNoFile = "../../testdata/trpc_go_error.yaml"
const configPath = "../../../testdata/router.yaml"

func Test_LoadConf(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	loader := ConfLoader{}
	err := loader.LoadConf(context.Background(), "fasthttp")
	assert.NotNil(t, err)
	DefaultRouterConfDir = "../../../testdata/router.d/"
	DefaultRouterConfFile = "../../../testdata/router.yaml"
	err = loader.LoadConf(context.Background(), "fasthttp")
	assert.NotNil(t, err)

	DefaultRouterConfDir = ""
	DefaultRouterConfFile = configPath
	mockRouter := mock_router.NewMockRouter(ctrl)
	mockRouter.EXPECT().InitRouterConfig(gomock.Any(), gomock.Any()).Return(nil)
	router.RegisterRouter("fasthttp", mockRouter)
	err = loader.LoadConf(context.Background(), "fasthttp")
	assert.Nil(t, err)

	_, err = getConfigFromFile(configPathNoFile)
	assert.NotNil(t, err)
	_, err = getConfigFromFile(configPathInvalid)
	assert.NotNil(t, err)
	_, err = getConfigFromFile(configPath)
	assert.Nil(t, err)
}

func Test_getConfigFromFile(t *testing.T) {
	_, err := getConfigFromFile(configPathNoFile)
	assert.NotNil(t, err)
	_, err = getConfigFromFile(configPathInvalid)
	assert.NotNil(t, err)
	_, err = getConfigFromFile(configPath)
	assert.Nil(t, err)
}
