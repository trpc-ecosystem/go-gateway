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

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-gateway/core/entity"
)

func TestProxyConfig(t *testing.T) {
	// load config
	confBytes, err := os.ReadFile("../../testdata/router.yaml")
	assert.Nil(t, err)
	var proxyConfig entity.ProxyConfig
	err = yaml.Unmarshal(confBytes, &proxyConfig)
	assert.Nil(t, err)
	config, _ := json.MarshalIndent(proxyConfig, "", "  ")
	t.Log(string(config))
	m, _ := yaml.Marshal(proxyConfig)
	fmt.Println(string(m))
}
