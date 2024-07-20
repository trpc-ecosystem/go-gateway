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

package fhttp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/valyala/fasthttp"
	gerrs "trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	ghttp "trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/core/config"
	reuseport "trpc.group/trpc-go/trpc-gateway/internal/reuseport"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/transport"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"
)

// ProtocolName fasthttp 协议
const ProtocolName = "fasthttp"

func init() {
	transport.RegisterClientTransport(ProtocolName, DefaultClientTransport)
	config.RegisterCustomTransOpts(ProtocolName, SetupCustomTransOpts)
}

var (
	// DefaultServerTransport is the default server HTTP transport.
	DefaultServerTransport = NewServerTransport(config.WithReusePort(true))
	// DefaultClientTransport is the default client HTTP transport.
	DefaultClientTransport = NewClientTransport()

	// defaultTimeout is the default timeout for backend services (500ms).
	defaultTimeout = time.Duration(500) * time.Millisecond
)

// ServerTransport is the HTTP transport layer.
type ServerTransport struct {
	Server *fasthttp.Server
	opts   *config.ServerOptions
}

// NewServerTransport creates a new HTTP transport.
func NewServerTransport(opt ...config.ServerOption) transport.ServerTransport {
	opts := &config.ServerOptions{}
	opts.IdleTimeout = time.Minute
	// Add the provided func options to the opts field
	for _, o := range opt {
		o(opts)
	}
	s := &ServerTransport{
		Server: &fasthttp.Server{},
		opts:   opts,
	}
	return s
}

// SetupCustomTransOpts sets up custom server options.
func SetupCustomTransOpts(opts ...config.ServerOption) {
	opts = append(opts, config.WithReusePort(true))
	DefaultServerTransport = NewServerTransport(opts...)
	transport.RegisterServerTransport(ProtocolName, DefaultServerTransport)
}

// ListenAndServe handles the configuration.
func (st *ServerTransport) ListenAndServe(ctx context.Context, opt ...transport.ListenServeOption) error {
	opts := &transport.ListenServeOptions{
		Network: "tcp",
	}

	for _, o := range opt {
		o(opts)
	}

	// Get the listener
	ln, err := st.getListener(opts)
	if err != nil {
		return gerrs.Wrap(err, "get_listener_err")
	}

	// Save the listener
	if err := transport.SaveListener(ln); err != nil {
		return gerrs.Wrap(err, "save fasthttp listener err")
	}

	// Configure TLS
	if len(opts.TLSKeyFile) != 0 && len(opts.TLSCertFile) != 0 {
		tlsConf, err := generateTLSConfig(opts)
		if err != nil {
			return gerrs.Wrap(err, "generate_tls_conf_err")
		}
		ln = tls.NewListener(ln, tlsConf)
	}

	return st.serve(ctx, ln, opts)

}

var emptyBuf []byte

func (st *ServerTransport) serve(ctx context.Context, ln net.Listener, opts *transport.ListenServeOptions) error {
	serveFunc := func(fctx *fasthttp.RequestCtx) {
		// Generate gateway message and save it to the context
		innerCtx, gMsg := gwmsg.WithNewGWMessage(context.Background())
		defer gwmsg.PutBackGwMessage(gMsg)
		// Generate a new empty message structure and save it to the context
		innerCtx = ghttp.WithRequestContext(innerCtx, fctx)
		innerCtx, msg := codec.WithNewMessage(innerCtx)
		defer codec.PutBackMessage(msg)

		msg.WithLocalAddr(fctx.LocalAddr())
		msg.WithRemoteAddr(fctx.RemoteAddr())
		msg.WithCallerServiceName(opts.ServiceName)
		msg.WithCallerMethod(string(fctx.Path()))
		_, err := opts.Handler.Handle(innerCtx, emptyBuf)
		if err != nil {
			log.ErrorContextf(innerCtx, "http server transport handle fail:%v", err)
			fctx.SetStatusCode(fasthttp.StatusInternalServerError)
		}
	}

	st.Server.Handler = serveFunc
	st.Server.Concurrency = st.opts.MaxCons
	st.Server.MaxConnsPerIP = st.opts.MaxConsPerIP
	st.Server.ReadTimeout = st.opts.ReadTimeout
	st.Server.WriteTimeout = st.opts.WriteTimeout
	st.Server.MaxRequestBodySize = st.opts.MaxRequestBodySize
	st.Server.ReadBufferSize = st.opts.ReadBufferSize
	st.Server.Name = ghttp.GatewayName

	go func() {
		_ = st.Server.Serve(ln)
	}()
	if st.opts.ReusePort {
		go func() {
			<-ctx.Done()
			_ = st.Server.Shutdown()
		}()
	}

	return nil
}

// GetPassedListener is renamed for stubbing in unit tests
var GetPassedListener = transport.GetPassedListener

// getListener gets the listener
func (st *ServerTransport) getListener(opts *transport.ListenServeOptions) (net.Listener, error) {
	var err error
	var ln net.Listener

	v, _ := os.LookupEnv(transport.EnvGraceRestart)
	// Convert environment variable to boolean, ignore error if conversion fails
	ok, _ := strconv.ParseBool(v)
	if ok {
		// Find the passed listener
		pln, gErr := GetPassedListener(opts.Network, opts.Address)
		if gErr != nil {
			return nil, gerrs.Wrap(gErr, "get_passed_listener_err")
		}

		ln, ok = pln.(net.Listener)
		if !ok {
			return nil, errors.New("invalid net.Listener")
		}

		return ln, nil
	}

	if st.opts.ReusePort {
		ln, err = reuseport.Listen(opts.Network, opts.Address)
		if err != nil {
			return nil, gerrs.Wrap(err, "fasthttp reuseport listen err")
		}
	} else {
		ln, err = net.Listen(opts.Network, opts.Address)
		if err != nil {
			return nil, gerrs.Wrap(err, "fasthttp listen err")
		}
	}

	return ln, nil
}

// generateTLSConfig generates TLS configuration
func generateTLSConfig(opts *transport.ListenServeOptions) (*tls.Config, error) {
	tlsConf := &tls.Config{}

	cert, err := tls.LoadX509KeyPair(opts.TLSCertFile, opts.TLSKeyFile)
	if err != nil {
		return nil, gerrs.Wrap(err, "LoadX509KeyPair_err")
	}
	tlsConf.Certificates = []tls.Certificate{cert}

	// Mutual authentication
	if opts.CACertFile != "" {
		tlsConf.ClientAuth = tls.RequireAndVerifyClientCert
		if opts.CACertFile != "root" {
			ca, rfErr := os.ReadFile(opts.CACertFile)
			if rfErr != nil {
				return nil, gerrs.Wrap(rfErr, "read_ca_cert_file_err")
			}
			pool := x509.NewCertPool()
			ok := pool.AppendCertsFromPEM(ca)
			if !ok {
				return nil, errors.New("failed to append certs from pem")
			}
			tlsConf.ClientCAs = pool
		}
	}
	SetCustomTLSOptions(tlsConf)
	return tlsConf, nil
}

// ClientTransport is the client-side HTTP transport.
type ClientTransport struct {
	opts     *transport.ClientTransportOptions
	connPoll Pool
}

// NewClientTransport creates a new HTTP transport.
func NewClientTransport(opt ...transport.ClientTransportOption) transport.ClientTransport {
	opts := &transport.ClientTransportOptions{}

	// Add the provided func options to the opts field
	for _, o := range opt {
		o(opts)
	}
	return &ClientTransport{
		opts:     opts,
		connPoll: NewConnPool(DefaultConnPoolSize),
	}
}

// DefaultClientReadBufferSize is the default buffer size for reading responses.
// It limits the size of response headers. Clients can override this configuration.
var DefaultClientReadBufferSize = 4096

// RoundTrip sends and receives HTTP requests and stores the HTTP response in the context.
// It does not need to return the response body.
func (ct *ClientTransport) RoundTrip(ctx context.Context, _ []byte,
	callOpts ...transport.RoundTripOption) (rspbody []byte, err error) {
	var opts transport.RoundTripOptions
	for _, o := range callOpts {
		o(&opts)
	}
	msg := codec.Message(ctx)
	fctx := ghttp.RequestContext(ctx)
	if fctx == nil {
		return nil, errs.NewFrameError(gerrs.ErrWrongContext, "client transport: fasthttp requestCtx")
	}
	// Set the target IP address
	tcpAddr, err := net.ResolveTCPAddr(opts.Network, opts.Address)
	if err != nil {
		// Only used for reporting, degrade gracefully
		log.ErrorContextf(ctx, "resolve upstream addr err:%s", err)
	}
	msg.WithRemoteAddr(tcpAddr)
	// Store in gwmsg for reporting
	gwmsg.GwMessage(ctx).WithUpstreamAddr(opts.Address)

	// Add request headers
	ct.setReqHead(fctx, msg)

	proxyClient, err := ct.connPoll.Get(opts.Address)
	if err != nil {
		return nil, gerrs.Wrap(err, "get_conn_err")
	}
	proxyClient.Addr = opts.Address
	defer func() {
		_ = ct.connPoll.Put(proxyClient)
	}()
	// Get the timeout
	timeout := msg.RequestTimeout()
	if timeout == 0 {
		timeout = defaultTimeout
	}
	proxyClient.MaxIdleConnDuration = timeout
	proxyClient.ReadTimeout = timeout
	proxyClient.WriteTimeout = timeout
	proxyClient.ReadBufferSize = DefaultClientReadBufferSize

	req := &fctx.Request
	resp := &fctx.Response
	// Do not set default content-type
	req.Header.SetNoDefaultContentType(true)

	start := time.Now()
	err = proxyClient.Do(req, resp)
	// Resolve the issue of closing the connection due to idle time when reusing the connection
	if err == fasthttp.ErrConnectionClosed {
		err = proxyClient.Do(req, resp)
	}
	resp.Header.SetNoDefaultContentType(true)

	if err != nil {
		if err == fasthttp.ErrTimeout {
			return nil, errs.NewFrameError(errs.RetClientTimeout,
				"http client transport RoundTrip timeout: "+err.Error())
		}

		if ctx.Err() == context.Canceled {
			return nil, errs.NewFrameError(errs.RetClientCanceled,
				"http client transport RoundTrip canceled: "+err.Error())
		}

		return nil, errs.NewFrameError(errs.RetClientNetErr, "http client transport RoundTrip: "+err.Error())
	}
	gwmsg.GwMessage(msg.Context()).WithUpstreamLatency(time.Since(start).Milliseconds())
	// Report exceptions for non-successful upstream response status codes
	if !gerrs.IsSuccessHTTPStatus(int32(resp.StatusCode())) {
		// TODO 如果有 trpc 的框架 err 信息返回，就返回 trpc err。不会是业务错误，业务错误会返回 200
		fmt.Println(string(resp.Header.Peek("trpc-error-msg")))
		fmt.Println(string(resp.Header.Peek("trpc-ret")))
		return nil, errs.New(gerrs.ErrUpstreamRspErr,
			fmt.Sprintf("upstream http status code:%v,msg", resp.StatusCode()))
	}
	return
}

// setReqHead sets the request headers
func (ct *ClientTransport) setReqHead(fctx *fasthttp.RequestCtx, msg codec.Msg) {
	fctx.Request.Header.Set(http.TrpcCaller, msg.CallerServiceName())
	fctx.Request.Header.Set(http.TrpcCallee, msg.CalleeServiceName())
	fctx.Request.Header.Set(http.TrpcTimeout, strconv.Itoa(int(msg.RequestTimeout()/time.Millisecond)))

	if len(msg.ClientMetaData()) > 0 {
		m := make(map[string]string)
		for k, v := range msg.ClientMetaData() {
			m[k] = base64.StdEncoding.EncodeToString(v)
		}

		// Set dyeing information
		if msg.Dyeing() {
			m[http.TrpcDyeingKey] = base64.StdEncoding.EncodeToString([]byte(msg.DyeingKey()))
			fctx.Request.Header.Set(http.TrpcMessageType, strconv.Itoa(int(trpcpb.TrpcMessageType_TRPC_DYEING_MESSAGE)))
		}

		m[http.TrpcEnv] = base64.StdEncoding.EncodeToString([]byte(msg.EnvTransfer()))
		val, _ := codec.Marshal(codec.SerializationTypeJSON, m)
		fctx.Request.Header.Set(http.TrpcTransInfo, string(val))
	}
}
