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

// Package file provides a router config loader with file.
package file

import (
	"context"
	"flag"
	"os"

	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/core/config"
	"trpc.group/trpc-go/trpc-gateway/core/entity"
	"trpc.group/trpc-go/trpc-gateway/core/router"
	"trpc.group/trpc-go/trpc-gateway/internal/util"
)

const fileConfProvider = "file_router"

var (
	// DefaultRouterConfFile is the default router configuration file, required
	DefaultRouterConfFile = "../conf/router.yaml"
	// DefaultRouterConfDir is the default router configuration directory, multiple files can be stored based on types,
	// if available, this directory is loaded first, not required
	DefaultRouterConfDir = "../conf/router.d/"
)

func init() {
	config.RegisterConfLoader(fileConfProvider, &ConfLoader{})

	flag.StringVar(&DefaultRouterConfFile, "router", DefaultRouterConfFile, "router conf file")
}

// ConfLoader is a file configuration loader
type ConfLoader struct {
}

// LoadConf loads the configuration
func (l *ConfLoader) LoadConf(ctx context.Context, protocol string) error {
	var rf entity.ProxyConfig
	// If there are configurations in the directory, try to read scattered configurations first
	files, err := os.ReadDir(DefaultRouterConfDir)
	if err != nil && !os.IsNotExist(err) {
		return errs.Wrap(err, "read dir err")
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		// If configured, ensure accuracy
		if err := loadAndAppend(DefaultRouterConfDir+f.Name(), &rf); err != nil {
			return errs.Wrapf(err, "load %s config err", DefaultRouterConfDir+f.Name())
		}
	}

	// Load the main configuration
	if err := loadAndAppend(DefaultRouterConfFile, &rf); err != nil {
		return errs.Wrapf(err, "load %s config err", DefaultRouterConfFile)
	}

	if err := router.GetRouter(protocol).InitRouterConfig(ctx, &rf); err != nil {
		return errs.Wrap(err, "init router from file err")
	}
	return nil
}

// loadAndAppend loads and appends the configuration
func loadAndAppend(filename string, config *entity.ProxyConfig) error {
	cfg, err := getConfigFromFile(filename)
	if err != nil {
		return errs.Wrap(err, "get_router_conf_err")
	}
	config.Router = append(config.Router, cfg.Router...)
	config.Client = append(config.Client, cfg.Client...)
	config.Plugins = append(config.Plugins, cfg.Plugins...)
	return nil
}

// getConfigFromFile retrieves the configuration from a file
func getConfigFromFile(fileName string) (*entity.ProxyConfig, error) {
	buf, err := os.ReadFile(fileName)
	if err != nil {
		return nil, errs.Wrapf(err, "read file [%s] failed", fileName)
	}

	var cfg entity.ProxyConfig
	buf = []byte(util.ExpandEnv(string(buf)))
	if err := yaml.Unmarshal(buf, &cfg); err != nil {
		return nil, errs.Wrapf(err, "unmarshal filer %s", fileName)
	}

	return &cfg, nil
}
