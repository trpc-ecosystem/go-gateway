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

// Package http converts fasthttp requests to net/http requests
// It refers to the logic of net/http/httputil/reverseproxy.go
package http

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"net"
	stdhttp "net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
	"golang.org/x/net/http/httpguts"
	"trpc.group/trpc-go/trpc-gateway/common/errs"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/core/service/protocol"
	"trpc.group/trpc-go/trpc-gateway/internal"
	"trpc.group/trpc-go/trpc-gateway/internal/pool/objectpool"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	terrs "trpc.group/trpc-go/trpc-go/errs"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/log"
)

// ProtocolHandler is the default protocol handler
type ProtocolHandler struct {
	// Director must be a function which modifies
	// the request into a new request to be sent
	// using Transport. Its response is then copied
	// back to the original client unmodified.
	// Director must not access the provided Request
	// after returning.
	Director func(*stdhttp.Request)

	// The transport used to perform proxy requests.
	// If nil, http.DefaultTransport is used.
	Transport stdhttp.RoundTripper

	// FlushInterval specifies the flush interval
	// to flush to the client while copying the
	// response body.
	// If zero, no periodic flushing is done.
	// A negative value means to flush immediately
	// after each write to the client.
	// The FlushInterval is ignored when ReverseProxy
	// recognizes a response as a streaming response, or
	// if its ContentLength is -1; for such responses, writes
	// are flushed to the client immediately.
	FlushInterval time.Duration

	// ErrorLog specifies an optional logger for errors
	// that occur when attempting to proxy the request.
	// If nil, logging is done via the log package's standard logger.

	// BufferPool optionally specifies a buffer pool to
	// get byte slices for use by io.CopyBuffer when
	// copying HTTP response bodies.
	BufferPool

	// ErrorHandler is an optional function that handles errors
	// reaching the backend or errors from ModifyResponse.
	//
	// If nil, the default is to log the provided error and return
	// a 502 Status Bad Gateway response.
	ErrorHandler func(context.Context, error)
}

// BufferPool is an interface for getting and returning temporary
// byte slices for use by io.CopyBuffer.
type BufferPool interface {
	Get() []byte
	Put([]byte)
}

func init() {
	protocol.RegisterCliProtocolHandler("http", DefaultProtocolHandler)
}

const defaultBufSize = 1024 * 32

var (
	// DefaultProtocolHandler 默认的协议转化处理器
	DefaultProtocolHandler = &ProtocolHandler{
		BufferPool: objectpool.NewBytesPool(defaultBufSize),
	}
)

// WithCtx sets the context
func (dph *ProtocolHandler) WithCtx(ctx context.Context) (context.Context, error) {
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return nil, terrs.Newf(errs.ErrWrongContext, "failed to get fctx")
	}

	outReq := &stdhttp.Request{}
	// Reset the path
	if err := convertRequest(fctx, outReq); err != nil {
		return nil, errs.Wrap(err, "failed to convert request")
	}
	if outReq.ContentLength == 0 {
		outReq.Body = nil // Issue 16036: nil Body for http.Transport retries
	}
	if outReq.Body != nil {
		// Reading from the request body after returning from a handler is not
		// allowed, and the RoundTrip goroutine that reads the Body can outlive
		// this handler. This can lead to a crash if the handler panics (see
		// Issue 46866). Although calling Close doesn't guarantee there isn't
		// any Read in flight after the handle returns, in practice it's safe to
		// read after closing it.
		defer outReq.Body.Close()
	}

	// Set keep-alive
	outReq.Close = false

	reqUpType := dph.upgradeType(outReq.Header)
	if !isPrint(reqUpType) {
		return nil, terrs.Newf(errs.ErrInvalidReq, "client tried to switch to invalid protocol %q", reqUpType)
	}
	dph.removeConnectionHeaders(outReq.Header)

	// Remove hop-by-hop headers to the backend. Especially
	// important is "Connection" because we want a persistent
	// connection, regardless of what the client sent to us.
	for _, h := range internal.HopHeaders {
		outReq.Header.Del(h)
	}

	// Issue 21096: tell backend applications that care about trailer support
	// that we support trailers. (We do, but we don't go out of our way to
	// advertise that unless the incoming client request thought it was worth
	// mentioning.) Note that we look at req.Header, not outReq.Header, since
	// the latter has passed through removeConnectionHeaders.
	// Convert to []string
	var teHeaders []string
	for _, v := range fctx.Request.Header.PeekAll("Te") {
		teHeaders = append(teHeaders, string(v))
	}
	if httpguts.HeaderValuesContainsToken(teHeaders, "trailers") {
		outReq.Header.Set("Te", "trailers")
	}

	// After stripping all the hop-by-hop connection headers above, add back any
	// necessary for protocol upgrades, such as for websockets.
	if reqUpType != "" {
		outReq.Header.Set("Connection", "Upgrade")
		outReq.Header.Set("Upgrade", reqUpType)
	}
	if reqUpType != "" || string(fctx.Request.Header.Peek("Accept-Encoding")) == "chunked" {
		// Copy ctx, remove timeout. Websockets and HTTP chunked do not set a timeout.
		//ctx = trpc.CloneContext(ctx)
	}

	header := &thttp.ClientReqHeader{
		Schema: string(fctx.Request.URI().Scheme()),
		Method: string(fctx.Method()),
		Host:   string(fctx.Host()),
		Header: outReq.Header,
	}
	// Reset the RPC name, handle cases with query parameters like /user?name=xxx
	codec.Message(ctx).WithClientRPCName(outReq.URL.RequestURI())
	codec.Message(ctx).WithClientReqHead(header)
	return ctx, nil
}

// TransReqBody converts the request body
func (dph *ProtocolHandler) TransReqBody(ctx context.Context) (interface{}, error) {
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return nil, nil
	}
	// Force conversion to an HTTP request
	fctx.Request.URI().SetScheme("http")
	return &codec.Body{Data: fctx.Request.Body()}, nil
}

// TransRspBody converts the response body
func (dph *ProtocolHandler) TransRspBody(context.Context) (interface{}, error) {
	return &codec.Body{}, nil
}

// GetCliOptions gets specific client options for the request
func (dph *ProtocolHandler) GetCliOptions(_ context.Context) ([]client.Option, error) {
	rspHead := &thttp.ClientRspHeader{
		ManualReadBody: true,
	}
	opts := []client.Option{
		client.WithRspHead(rspHead),
		client.WithTimeout(0),
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithCurrentCompressType(codec.CompressTypeNoop),
	}
	return opts, nil
}

// removeConnectionHeaders removes hop-by-hop headers listed in the "Connection" header of h.
// See RFC 7230, section 6.1
func (dph *ProtocolHandler) removeConnectionHeaders(h stdhttp.Header) {
	for _, f := range h["Connection"] {
		for _, sf := range strings.Split(f, ",") {
			if sf = textproto.TrimString(sf); sf != "" {
				h.Del(sf)
			}
		}
	}
}

// Get the upgrade type
func (dph *ProtocolHandler) upgradeType(h stdhttp.Header) string {
	if !httpguts.HeaderValuesContainsToken(h["Connection"], "Upgrade") {
		return ""
	}
	return h.Get("Upgrade")
}

// HandleErr handles error messages
func (dph *ProtocolHandler) HandleErr(ctx context.Context, err error) error {
	if err == nil {
		return err
	}
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return err
	}

	var te *terrs.Error
	if ok := errors.As(err, &te); !ok {
		return err
	}

	if te.Type == terrs.ErrorTypeBusiness {
		// Business error code, indicates successful forwarding, do not return err
		dph.HandleRspBody(ctx, nil)
		fctx.Response.Header.Set(thttp.TrpcErrorMessage, terrs.Msg(err))
		fctx.Response.Header.Set(thttp.TrpcUserFuncErrorCode, strconv.Itoa(int(terrs.Code(err))))
		return nil
	}
	return err
}

// HandleRspBody handles the response
func (dph *ProtocolHandler) HandleRspBody(ctx context.Context, _ interface{}) error {
	msg := codec.Message(ctx)
	rspHeader, ok := msg.ClientRspHead().(*thttp.ClientRspHeader)
	if !ok {
		return terrs.New(errs.ErrWrongContext, "failed to get thttp response header")
	}
	reqHeader, ok := msg.ClientReqHead().(*thttp.ClientReqHeader)
	if !ok {
		return terrs.New(errs.ErrWrongContext, "failed to get thttp request header")
	}
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return terrs.New(errs.ErrWrongContext, "failed to get fasthttp context")
	}
	// Deal with 101 Switching Protocols responses: (WebSocket, h2c, etc)
	if rspHeader.Response.StatusCode == stdhttp.StatusSwitchingProtocols {
		if err := dph.handleUpgradeResponse(ctx, reqHeader.Header, rspHeader.Response); err != nil {
			return errs.Wrap(err, "failed to handle upgrade response")
		}
		return nil
	}
	dph.removeConnectionHeaders(rspHeader.Response.Header)
	for _, h := range internal.HopHeaders {
		rspHeader.Response.Header.Del(h)
	}
	dph.copyHeader2fctxRspHeader(fctx, rspHeader.Response.Header)

	// The "Trailer" header isn't included in the Transport's response,
	// at least for *http.Transport. Build it up from Trailer.
	announcedTrailers := len(rspHeader.Response.Trailer)
	if announcedTrailers > 0 {
		trailerKeys := make([]string, 0, len(rspHeader.Response.Trailer))
		for k := range rspHeader.Response.Trailer {
			trailerKeys = append(trailerKeys, k)
		}
		fctx.Response.Header.Add("Trailer", strings.Join(trailerKeys, ", "))
	}
	fctx.Response.SetStatusCode(rspHeader.Response.StatusCode)
	flushInterval := dph.flushInterval(rspHeader.Response)
	if flushInterval != -1 {
		fctx.Response.SetBodyStream(rspHeader.Response.Body, int(rspHeader.Response.ContentLength))
		return nil
	}
	fctx.Response.ImmediateHeaderFlush = true
	fctx.Response.SetBodyStreamWriter(func(w *bufio.Writer) {
		err := dph.copyResponse(ctx, w, rspHeader.Response.Body, flushInterval)
		if err != nil {
			defer rspHeader.Response.Body.Close()
			// Since we're streaming the response, if we run into an error all we can do is abort the request.
			// Issue 23643: ReverseProxy should use ErrAbortHandler on read error while copying body.
			//if !shouldPanicOnCopyError(req) {
			//	p.logf
			//	("suppressing panic for copyResponse error in test; copy error: %v", err)
			//	return
			//}
			log.ErrorContextf(ctx, "copy response err:%s", err)
			return
		}
		_ = rspHeader.Response.Body.Close() // close now, instead of defer, to populate res.Trailer
		if len(rspHeader.Response.Trailer) > 0 {
			// Force chunking if we saw a response trailer.
			// This prevents net/http from calculating the length for short
			// bodies and adding a Content-Length.
			if err := w.Flush(); err != nil {
				log.ErrorContextf(ctx, "flush trailer before body err ")
			}
		}
		if len(rspHeader.Response.Trailer) == announcedTrailers {
			dph.copyHeader2fctxRspHeader(fctx, rspHeader.Response.Trailer)
			return
		}
		for k, vv := range rspHeader.Response.Trailer {
			k = stdhttp.TrailerPrefix + k
			for _, v := range vv {
				fctx.Response.Header.Add(k, v)
			}
		}
	})
	return nil
}

// Handle the upgrade response
func (dph *ProtocolHandler) handleUpgradeResponse(ctx context.Context, reqHeader stdhttp.Header,
	res *stdhttp.Response) error {
	fctx := http.RequestContext(ctx)
	if fctx == nil {
		return nil
	}

	reqUpType := dph.upgradeType(reqHeader)
	resUpType := dph.upgradeType(res.Header)
	if !isPrint(resUpType) { // We know reqUpType is ASCII, it's checked by the caller.
		return terrs.Newf(errs.ErrInvalidReq, "backend tried to switch to invalid protocol %q", resUpType)
	}
	if !equalFold(reqUpType, resUpType) {
		return terrs.Newf(errs.ErrInvalidReq, "backend tried to switch protocol %q when %q was requested",
			resUpType, reqUpType)
	}
	fctx.Hijack(func(conn net.Conn) {
		dph.connProxy(ctx, conn, res)
	})
	fctx.SetStatusCode(res.StatusCode)
	dph.copyHeader2fctxRspHeader(fctx, res.Header)
	return nil
}

// TestConnProxy is a variable that holds the connProxy function for testing purposes
var TestConnProxy = (&ProtocolHandler{}).connProxy

func (dph *ProtocolHandler) connProxy(ctx context.Context, conn net.Conn, res *stdhttp.Response) {
	backConn, ok := res.Body.(io.ReadWriteCloser)
	if !ok {
		log.ErrorContextf(ctx, "internal error: 101 switching protocols response with non-writable body")
		return
	}
	backConnCloseCh := make(chan bool)
	go func() {
		// Ensure that the cancellation of a request closes the backend.
		// See issue https://golang.org/issue/35559.
		select {
		// TODO: Figure out how to handle this, can't get the original request's request
		//case <-req.Context().Done():
		case <-backConnCloseCh:
			log.DebugContextf(ctx, "backConnCloseCh done")
		}
		_ = backConn.Close()
	}()

	defer close(backConnCloseCh)

	defer conn.Close()
	errc := make(chan error, 1)
	spc := switchProtocolCopier{user: conn, backend: backConn}
	go spc.copyToBackend(errc)
	go spc.copyFromBackend(errc)
	<-errc
}

// switchProtocolCopier exists so goroutines proxying data back and forth have nice names in stacks.
type switchProtocolCopier struct {
	user, backend io.ReadWriter
}

func (c switchProtocolCopier) copyToBackend(errc chan<- error) {
	_, err := io.Copy(c.backend, c.user)
	errc <- err
}

func (c switchProtocolCopier) copyFromBackend(errc chan<- error) {
	_, err := io.Copy(c.user, c.backend)
	errc <- err
}

// Copy net/http response headers to fasthttp
func (dph *ProtocolHandler) copyHeader2fctxRspHeader(dst *fasthttp.RequestCtx, src stdhttp.Header) {
	if src == nil {
		return
	}
	for k, vs := range src {
		for _, v := range vs {
			dst.Response.Header.Add(k, v)
		}
	}
}

// flushInterval returns the p.FlushInterval value, conditionally
// overriding its value for a specific request/response.
func (dph *ProtocolHandler) flushInterval(res *stdhttp.Response) (i time.Duration) {
	resCT := res.Header.Get("Content-Type")
	// For Server-Sent Events responses, flush immediately.
	// The MIME type is defined in https://www.w3.org/TR/eventsource/#text-event-stream
	if baseCT, _, _ := mime.ParseMediaType(resCT); baseCT == "text/event-stream" {
		return -1 // negative means immediately
	}

	// We might have the case of streaming for which Content-Length might be unset.
	if res.ContentLength == -1 {
		return -1
	}
	return dph.FlushInterval
}

func (dph *ProtocolHandler) copyResponse(ctx context.Context, dst io.Writer, src io.Reader, flushInterval time.Duration) error {
	if flushInterval != 0 {
		if wf, ok := dst.(writeFlusher); ok {
			mlw := &maxLatencyWriter{
				dst:     wf,
				latency: flushInterval,
			}
			defer mlw.stop()

			// set up initial timer so headers get flushed even if body writes are delayed
			mlw.flushPending = true
			mlw.t = time.AfterFunc(flushInterval, mlw.delayedFlush)

			dst = mlw
		}
	}

	var buf []byte
	if dph.BufferPool != nil {
		buf = dph.BufferPool.Get()
		defer dph.BufferPool.Put(buf)
	}
	_, err := dph.copyBuffer(ctx, dst, src, buf)
	if err != nil {
		return errs.Wrap(err, "copy buffer error")
	}
	return nil
}

type writeFlusher interface {
	io.Writer
	Flush() error
}

type maxLatencyWriter struct {
	dst          writeFlusher
	latency      time.Duration // non-zero; negative means to flush immediately
	mu           sync.Mutex    // protects t, flushPending, and dst.Flush
	t            *time.Timer
	flushPending bool
}

func (m *maxLatencyWriter) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n, err = m.dst.Write(p)
	if err != nil {
		return n, errs.Wrapf(err, "dst write error")
	}
	if m.latency < 0 {
		if err := m.dst.Flush(); err != nil {
			return n, errs.Wrap(err, "dst flush error")
		}
		return
	}
	if m.flushPending {
		return
	}
	if m.t == nil {
		m.t = time.AfterFunc(m.latency, m.delayedFlush)
	} else {
		m.t.Reset(m.latency)
	}
	m.flushPending = true
	return
}

func (m *maxLatencyWriter) delayedFlush() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.flushPending { // if stop was called but AfterFunc already started this goroutine
		return
	}
	m.dst.Flush()
	m.flushPending = false
}

func (m *maxLatencyWriter) stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.flushPending = false
	if m.t != nil {
		m.t.Stop()
	}
}

// copyBuffer returns any write errors or non-EOF read errors, and the amount of bytes written.
func (dph *ProtocolHandler) copyBuffer(ctx context.Context, dst io.Writer, src io.Reader, buf []byte) (int64, error) {
	if len(buf) == 0 {
		buf = make([]byte, 32*1024)
	}
	var written int64
	for {
		nr, rerr := src.Read(buf)
		if rerr != nil && rerr != io.EOF && rerr != context.Canceled {
			log.ErrorContextf(ctx, "httputil: ReverseProxy read error during body copy: %v\n", rerr)
		}
		if nr > 0 {
			nw, werr := dst.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if werr != nil {
				return written, errs.Wrap(werr, "write buffer error")
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				rerr = nil
			} else {
				rerr = errs.Wrap(rerr, "read buffer error")
			}
			return written, rerr
		}
	}
}

// ConvertRequest converts a fasthttp.Request to an http.Request.
// Reference: https://github.com/valyala/fasthttp/blob/master/fasthttpadaptor/request.go
func convertRequest(ctx *fasthttp.RequestCtx, r *stdhttp.Request) error {
	body := ctx.PostBody()
	strRequestURI := string(ctx.Request.RequestURI())

	rURL, err := url.ParseRequestURI(strRequestURI)
	if err != nil {
		return err
	}

	r.Method = string(ctx.Method())
	r.Proto = string(ctx.Request.Header.Protocol())
	if r.Proto == "HTTP/2" {
		r.ProtoMajor = 2
	} else {
		r.ProtoMajor = 1
	}
	r.ProtoMinor = 1
	r.ContentLength = int64(len(body))
	r.RemoteAddr = ctx.RemoteAddr().String()
	r.Host = string(ctx.Host())
	r.TLS = ctx.TLSConnectionState()
	r.Body = io.NopCloser(bytes.NewReader(body))
	r.URL = rURL

	if r.Header == nil {
		r.Header = make(stdhttp.Header)
	} else if len(r.Header) > 0 {
		for k := range r.Header {
			delete(r.Header, k)
		}
	}

	ctx.Request.Header.VisitAll(func(k, v []byte) {
		sk := string(k)
		sv := string(v)

		switch sk {
		case "Transfer-Encoding":
			r.TransferEncoding = append(r.TransferEncoding, sv)
		default:
			r.Header.Set(sk, sv)
		}
	})

	return nil
}
