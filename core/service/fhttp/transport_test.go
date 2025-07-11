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

package fhttp

import (
	"context"
	"encoding/base64"
	"errors"
	"net"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prashantv/gostub"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	ghttp "trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/core/config"
	"trpc.group/trpc-go/trpc-gateway/core/service/fhttp/mock"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/transport"
)

func TestNewServerTransport(t *testing.T) {
	opts := []config.ServerOption{
		config.WithReusePort(true),
	}

	s := NewServerTransport(opts...)
	assert.NotNil(t, s)
}

func TestSetupCustomTransOpts(t *testing.T) {
	opts := []config.ServerOption{
		config.WithReusePort(true),
	}
	SetupCustomTransOpts(opts...)
	assert.NotNil(t, DefaultServerTransport)
}

// Mock handler
type mockHandler struct{}

func (*mockHandler) Handle(context.Context, []byte) (rsp []byte, err error) {
	return nil, errors.New("err")
}

type netError struct {
	error
}

type fakeListen struct{}

func (c *fakeListen) Accept() (net.Conn, error) {
	return nil, &netError{errors.New("network failure")}
}
func (c *fakeListen) Close() error {
	return nil
}

func (c *fakeListen) Addr() net.Addr {
	return nil
}

func TestServerTransport_ListenAndServe(t *testing.T) {
	tp := DefaultServerTransport
	opts := []transport.ListenServeOption{
		transport.WithListenAddress("localhost:8080"),
		transport.WithListenNetwork("tcp"),
		transport.WithHandler(&mockHandler{}),
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	stub := gostub.New()
	defer stub.Reset()
	// Normal startup - without reusing port
	err := tp.ListenAndServe(context.Background(), opts...)
	assert.Nil(t, err)
	// Execute the fasthttp handler
	tp.(*ServerTransport).Server.Handler(&fasthttp.RequestCtx{})
	// Normal startup - with reusing port
	reuseSt := NewServerTransport(config.WithReusePort(false))
	opts = append(opts, transport.WithListenAddress("localhost:8082"))
	err = reuseSt.ListenAndServe(context.Background(), opts...)
	assert.Nil(t, err)

	// Graceful restart - success
	_ = os.Setenv(transport.EnvGraceRestart, "true")
	stub.StubFunc(&GetPassedListener, &fakeListen{}, nil)
	err = tp.ListenAndServe(context.Background(), opts...)
	assert.Nil(t, err)
	// Graceful restart - failure - failedto get listener
	stub.StubFunc(&GetPassedListener, &fakeListen{}, errors.New("get listener err"))
	err = tp.ListenAndServe(context.Background(), opts...)
	assert.NotNil(t, err)
	// Graceful restart - failure - listener exception
	stub.StubFunc(&GetPassedListener, fakeListen{}, nil)
	err = tp.ListenAndServe(context.Background(), opts...)
	assert.NotNil(t, err)
}

func TestNewClientTransport(t *testing.T) {
	opts := []transport.ClientTransportOption{
		func(*transport.ClientTransportOptions) {},
	}

	ct := NewClientTransport(opts...)
	assert.NotNil(t, ct)
}
func TestClientTransport_RoundTrip(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	stub := gostub.New()
	defer stub.Reset()
	//codec.RegisterSerializer(codec.SerializationTypeJSON, &codec.JSONPBSerialization{})
	ctx := trpc.BackgroundContext()

	// clientTransport := DefaultClientTransport
	mockPool := mock.NewMockPool(ctrl)
	clientTransport := &ClientTransport{
		connPoll: mockPool,
	}

	c := &fasthttp.HostClient{
		Addr: "qq.com",
		// Proxy only makes one request, no retries allowed
		MaxIdemponentCallAttempts: 1,
	}
	mockPool.EXPECT().Get(gomock.Any()).Return(c, nil).AnyTimes()
	mockPool.EXPECT().Put(gomock.Any()).Return(nil).AnyTimes()

	opts := []transport.RoundTripOption{
		transport.WithDialAddress("qq.com"),
	}

	_, err := clientTransport.RoundTrip(ctx, nil, opts...)
	assert.NotNil(t, err)

	fctx := &fasthttp.RequestCtx{}
	ctx = ghttp.WithRequestContext(ctx, fctx)
	ctx, msg := codec.WithNewMessage(ctx)
	_, err = clientTransport.RoundTrip(ctx, nil, opts...)
	assert.NotNil(t, err)

	fctx.Request.SetHost("qq.com")

	m := map[string][]byte{
		http.TrpcEnv: []byte(base64.StdEncoding.EncodeToString([]byte("development"))),
	}
	msg.WithClientMetaData(m)
	msg.WithDyeing(true)
	_, err = clientTransport.RoundTrip(ctx, nil, opts...)
	assert.NotNil(t, err)
}

func Test_generateTLSConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	//stub.StubFunc(&LoadX509KeyPair, tls.Certificate{}, nil)
	// Get TLS configuration successfully
	tlsConf, err := generateTLSConfig(&transport.ListenServeOptions{
		Network:     "tcp",
		CACertFile:  "../../../testdata/server.crt",
		TLSKeyFile:  "../../../testdata/server.key",
		TLSCertFile: "../../../testdata/server.crt",
	})
	assert.NotNil(t, tlsConf)
	assert.Nil(t, err)

	// Failed to get TLS configuration
	tlsConf, err = generateTLSConfig(&transport.ListenServeOptions{
		Network:     "tcp",
		CACertFile:  "../../../testdata/server.crt",
		TLSKeyFile:  "../../../testdata/server.key.not.exist",
		TLSCertFile: "../../../testdata/server.crt",
	})
	assert.NotNil(t, err)

	// Failed to get TLS configuration
	tlsConf, err = generateTLSConfig(&transport.ListenServeOptions{
		Network:     "tcp",
		CACertFile:  "../../../testdata/server.crt.not.exist",
		TLSKeyFile:  "../../../testdata/server.key",
		TLSCertFile: "../../../testdata/server.crt",
	})
	assert.NotNil(t, err)
}
