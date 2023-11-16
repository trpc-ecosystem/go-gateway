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

// Package config provides gateway configuration related functions
package config

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"go.uber.org/automaxprocs/maxprocs"
	"gopkg.in/yaml.v3"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/internal/util"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/server"
)

const defaultConfigPath = "./trpc_go.yaml"

func init() {
	trpc.ServerConfigPath = defaultConfigPath
	flag.StringVar(&trpc.ServerConfigPath, "conf", defaultConfigPath, "server config path")
}

// Config is the implementation of trpc configuration, divided into four major sections: global configuration (global),
// server configuration (server), client configuration (client), and plugin configuration (plugins).
type Config struct {
	Global struct {
		ConfProvider string `yaml:"conf_provider"` // Routing configuration provider, file, etcd etc.
	}
	Server struct {
		Service []*ServiceConfig // Configuration of a single service
	}
}

// ServiceConfig is the configuration for each service. A single service process can support multiple services.
type ServiceConfig struct {
	IP       string   `yaml:"ip"` // IP address to listen to.
	Name     string   // Service name in the format: trpc.app.server.service. Used for naming the service.
	Nic      string   // Network Interface Card (NIC) to listen to. No need to configure.
	Port     uint16   // Port to listen to.
	Address  string   // Address to listen to. If set, ipport will be ignored. Otherwise, ipport will be used.
	Network  string   // Network type like tcp/udp.
	Protocol string   // Protocol type like trpc.
	Timeout  int      // Longest time in milliseconds for a handler to handle a request.
	Idletime int      // Maximum idle time in milliseconds for a server connection. Default is 1 minute.
	Registry string   // Registry to use, e.g., polaris.
	Filter   []string // Filters for the service.

	// additional configuration settings for the trpc framework.
	// MaxCons refers to the maximum number of allowed connections for a service.
	MaxCons int `yaml:"max_cons"`
	// MaxConsPerIP refers to the maximum number of allowed connections per individual IP address.
	MaxConsPerIP int `yaml:"max_cons_per_ip"`
	// MaxRequestBodySize refers to the maximum size or volume allowed for the request body in a request.
	MaxRequestBodySize string `yaml:"max_request_body_size"`
	// ReadBufferSize  refers to the size or capacity of the read buffer,
	// which is used for storing incoming data during the reading process.
	ReadBufferSize string `yaml:"read_buffer_size"`
}

// NewServer parses the yaml config file to quickly start the server with multiple services.
// The config file is ./trpc_go.yaml by default and can be set by the flag -conf.
// This method should be called only once.
func NewServer(opt ...server.Option) *server.Server {
	// load and parse config file
	cfg, err := trpc.LoadConfig(trpc.ServerConfigPath)
	if err != nil {
		panic("load config fail: " + err.Error())
	}

	// set to global config for other plugins' accessing to the config
	trpc.SetGlobalConfig(cfg)

	if sErr := trpc.Setup(cfg); sErr != nil {
		panic("setup plugin fail: " + sErr.Error())
	}

	err = setup()
	if err != nil {
		panic(err)
	}

	// set default GOMAXPROCS for docker
	_, _ = maxprocs.Set(maxprocs.Logger(log.Debugf))
	return trpc.NewServerWithConfig(cfg, opt...)
}

// setup parses the configuration related to the gateway.
func setup() error {
	cfg, err := loadConf(trpc.ServerConfigPath)
	if err != nil {
		return gerrs.Wrap(err, "load_config_fail")
	}

	// load the configuration for each service.
	for _, conf := range cfg.Server.Service {
		if conf.Protocol != "fasthttp" {
			return errors.New("unsupported protocol: " + conf.Protocol)
		}
		bodySize, _ := bytefmt.ToBytes(conf.MaxRequestBodySize)
		bufSize, _ := bytefmt.ToBytes(conf.ReadBufferSize)

		opts := []ServerOption{
			WithReadTimeout(time.Duration(conf.Timeout) * time.Millisecond),
			WithWriteTimeout(time.Duration(conf.Timeout) * time.Millisecond),
			WithMaxCons(conf.MaxCons),
			WithMaxRequestBodySize(int(bodySize)),
			WithReadBufferSize(int(bufSize)),
			WithMaxConsPerIP(conf.MaxConsPerIP),
			WithReusePort(true),
		}

		// set custom service options.
		if o := GetCustomTransOpts(conf.Protocol); o != nil {
			o(opts...)
		}

		loadRouter := GetRouteLoader(conf.Protocol)
		if loadRouter == nil {
			return fmt.Errorf("router loader [%s] not register", conf.Protocol)
		}
		err = loadRouter(cfg.Global.ConfProvider)
		if err != nil {
			log.ErrorContextf(context.Background(), "load router config err:%s", err)
			return gerrs.Wrap(err, "load router config err")
		}
	}

	return nil
}

var loadConf = func(configPath string) (*Config, error) {
	cfg := &Config{}

	buf, err := os.ReadFile(configPath)
	if err != nil {
		return nil, gerrs.Wrap(err, "read_config_err")
	}
	// parse environment variables.
	buf = []byte(util.ExpandEnv(string(buf)))

	if err := yaml.Unmarshal(buf, cfg); err != nil {
		return nil, gerrs.Wrap(err, "unmarshal_cfg_err")
	}

	// set the default config provider
	if cfg.Global.ConfProvider == "" {
		cfg.Global.ConfProvider = "file"
	}
	return cfg, nil
}
