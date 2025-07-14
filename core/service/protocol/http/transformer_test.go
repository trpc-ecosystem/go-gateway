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

// Reverse proxy tests.
package http_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	stdhttp "net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	chttp "trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/core/service/protocol/http"
	"trpc.group/trpc-go/trpc-gateway/internal/pool/objectpool"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	thttp "trpc.group/trpc-go/trpc-go/http"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"
)

func TestProtocolHandler_TransReqBody(t *testing.T) {
	dph := http.DefaultProtocolHandler
	got, err := dph.TransReqBody(context.Background())
	assert.Nil(t, err)
	assert.Nil(t, got)

	fctx := &fasthttp.RequestCtx{}
	fctx.Request.SetBodyString("xxx")
	ctx := chttp.WithRequestContext(context.Background(), fctx)
	b, err := dph.TransReqBody(ctx)
	assert.Nil(t, err)
	assert.Equal(t, []byte("xxx"), b.(*codec.Body).Data)
	rspBody, err := dph.TransRspBody(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, (&codec.Body{}).Data, rspBody.(*codec.Body).Data)
}

func TestProtocolHandler_GetCliOptions(t *testing.T) {
	dph := http.DefaultProtocolHandler
	opts, err := dph.GetCliOptions(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, 5, len(opts))
}

func TestProtocolHandler_HandleErr(t *testing.T) {

	dph := &http.ProtocolHandler{}
	err := dph.HandleErr(context.Background(), nil)
	assert.Nil(t, err)

	err = dph.HandleErr(context.Background(), errs.New(444, "err"))
	assert.Equal(t, trpcpb.TrpcRetCode(444), errs.Code(err))

	fctx := &fasthttp.RequestCtx{}
	ctx := chttp.WithRequestContext(context.Background(), fctx)
	err = dph.HandleErr(ctx, errors.New("err"))
	assert.Equal(t, "err", err.Error())

	err = dph.HandleErr(ctx, &errs.Error{
		Type: errs.ErrorTypeBusiness,
		Code: 1,
		Msg:  "errr",
		Desc: "",
	})
	assert.Nil(t, err)
	assert.Equal(t, []byte("1"), fctx.Response.Header.Peek(thttp.TrpcUserFuncErrorCode))

	err = dph.HandleErr(ctx, &errs.Error{
		Type: errs.ErrorTypeFramework,
		Code: 1,
		Msg:  "errr",
		Desc: "",
	})
	assert.NotNil(t, err)
}

func TestProtocolHandler_HandleRspBody(t *testing.T) {
	dph := http.ProtocolHandler{
		FlushInterval: 10,
		BufferPool:    objectpool.NewBytesPool(100),
	}
	err := dph.HandleRspBody(context.Background(), nil)
	assert.NotNil(t, err)
	ctx, msg := codec.WithNewMessage(context.Background())
	stdRsp := &stdhttp.Response{}
	stdRsp.Header = map[string][]string{}
	msg.WithClientRspHead(&thttp.ClientRspHeader{
		// TODO 等着 wineguo 给 merge 到 opensource 分支
		// ManualReadBody: true,
		Response: stdRsp,
	})
	err = dph.HandleRspBody(ctx, nil)
	assert.NotNil(t, err)

	msg.WithClientReqHead(&thttp.ClientReqHeader{})
	err = dph.HandleRspBody(ctx, nil)
	assert.NotNil(t, err)

	msg.WithClientReqHead(&thttp.ClientReqHeader{})
	fctx := &fasthttp.RequestCtx{}
	ctx = chttp.WithRequestContext(ctx, fctx)
	stdRsp.Trailer = map[string][]string{"Te": {"sss"}}

	stdRsp.StatusCode = stdhttp.StatusSwitchingProtocols
	stdRsp.Header.Set("Connection", "Upgrade")
	stdRsp.Header.Set("Upgrade", "Gophers like 🧀")
	err = dph.HandleRspBody(ctx, nil)
	assert.NotNil(t, err)

	clientReqHeader := &thttp.ClientReqHeader{}
	msg.WithClientReqHead(clientReqHeader)

	clientReqHeader.Header = map[string][]string{}
	clientReqHeader.Header.Set("Connection", "Upgrade")
	clientReqHeader.Header.Set("Upgrade", "h2c")
	stdRsp.Header.Set("Upgrade", "websocket")

	err = dph.HandleRspBody(ctx, nil)
	assert.NotNil(t, err)

	clientReqHeader.Header.Set("Connection", "Upgrade")
	clientReqHeader.Header.Set("Upgrade", "websocket")
	stdRsp.Header.Set("Upgrade", "websocket")
	err = dph.HandleRspBody(ctx, nil)
	assert.Nil(t, err)

	stdRsp.StatusCode = stdhttp.StatusOK
	err = dph.HandleRspBody(ctx, nil)
	assert.Nil(t, err)

	stdRsp.ContentLength = -1
	stdRsp.Body = io.NopCloser(strings.NewReader("ssssssssssssss"))
	err = dph.HandleRspBody(ctx, nil)
	assert.Nil(t, err)

	stdRsp.Header.Set("Content-Type", "text/event-stream")
	stdRsp.Body = io.NopCloser(strings.NewReader("ssssssssssssss"))
	err = dph.HandleRspBody(ctx, nil)
	assert.Nil(t, err)

	stdRsp.ContentLength = -1
	stdRsp.Body = io.NopCloser(strings.NewReader("ssssssssssssss"))
	err = dph.HandleRspBody(ctx, nil)
	assert.Nil(t, err)
}

func TestProtocolHandler_WithCtx(t *testing.T) {
	dph := &http.ProtocolHandler{}
	_, err := dph.WithCtx(context.Background())
	assert.NotNil(t, err)

	fctx := &fasthttp.RequestCtx{}
	fctx.Request.SetBodyString("xxxx")
	ctx := chttp.WithRequestContext(context.Background(), fctx)
	ctx, msg := codec.WithNewMessage(ctx)
	_, err = dph.WithCtx(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, msg.ClientReqHead())

	fctx.Request.SetBodyString("")
	fctx.Request.Header.Set("Te", "trailers")
	fctx.Request.Header.Set("Connection", "Upgrade")
	fctx.Request.Header.Set("Upgrade", "websocket")
	fctx.Request.Header.Set(fasthttp.HeaderXForwardedFor, "127.0.0.1")
	ctx, msg = codec.WithNewMessage(ctx)
	_, err = dph.WithCtx(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, msg.ClientReqHead())

	fctx.Request.Header.Set("Upgrade", "Gophers like 🧀")
	ctx, msg = codec.WithNewMessage(ctx)
	_, err = dph.WithCtx(ctx)
	assert.NotNil(t, err)

	fctx.Request.Header.Set("Upgrade", "websocket")
	fctx.Request.SetRequestURI("foo.html")
	ctx, msg = codec.WithNewMessage(ctx)
	_, err = dph.WithCtx(ctx)
	assert.NotNil(t, err)

	fctx.Request.SetRequestURI("/foo.html")
	ctx, msg = codec.WithNewMessage(ctx)
	_, err = dph.WithCtx(ctx)
	assert.Nil(t, err)
}

type readWriter struct {
	net.Conn
	r bytes.Buffer
	w bytes.Buffer
}

type readerWriterCloser struct {
}

func (*readerWriterCloser) Write(_ []byte) (int, error) {
	return 0, nil
}
func (*readerWriterCloser) Read(_ []byte) (int, error) {
	return 0, nil
}

func (*readerWriterCloser) Close() error {
	return nil
}

func TestConnProxy(t *testing.T) {
	stdRsp := &stdhttp.Response{}
	stdRsp.Body = &readerWriterCloser{}
	rw := new(readWriter)
	rw.Conn = &net.TCPConn{}
	rw.r.WriteString("GET /foo HTTP/1.1\r\nHost: google.com\r\n\r\n")
	http.TestConnProxy(context.Background(), rw, stdRsp)

	stdRsp.Body = io.NopCloser(strings.NewReader("xxx"))
	http.TestConnProxy(context.Background(), rw, stdRsp)
}

type copyBufferWriterCloser struct {
	buf []byte
}

func (cb *copyBufferWriterCloser) Write(p []byte) (int, error) {
	cb.buf = append(cb.buf, p...)
	return len(p), nil
}
func (cb *copyBufferWriterCloser) Read(p []byte) (int, error) {
	if len(cb.buf) == 0 {
		return 0, io.EOF
	}
	copy(p[0:len(cb.buf)], cb.buf)
	len := len(cb.buf)
	cb.buf = nil
	return len, nil
}

func (*copyBufferWriterCloser) Close() error {
	return nil
}

type Buffer struct {
	bytes.Buffer
	io.ReaderFrom // conflicts with and hides bytes.Buffer's ReaderFrom.
	io.WriterTo   // conflicts with and hides bytes.Buffer's WriterTo.
}

func TestProtocolHandler_copyBuffer(t *testing.T) {
	user := new(copyBufferWriterCloser)
	backend := new(copyBufferWriterCloser)

	backend.Write([]byte("123"))
	exitChannel := make(chan struct{}, 2)

	exitChannel <- struct{}{}
	http.TestCoyBuffer(user, backend, nil, exitChannel)

	assert.Nil(t, user.buf)
	assert.NotNil(t, <-exitChannel)
	assert.Equal(t, len(exitChannel), 0)

	http.TestCoyBuffer(user, backend, nil, exitChannel)
	assert.NotNil(t, user.buf)
}

func TestCopyReadFrom(t *testing.T) {
	rb := new(Buffer)
	wb := new(bytes.Buffer) // implements ReadFrom.
	rb.WriteString("hello, world.")
	http.TestCoyBuffer(wb, rb, nil, make(chan struct{}, 2))
	if wb.String() != "hello, world." {
		t.Errorf("Copy did not work properly")
	}
}

func TestCopyWriteTo(t *testing.T) {
	rb := new(bytes.Buffer) // implements WriteTo.
	wb := new(Buffer)
	rb.WriteString("hello, world.")
	http.TestCoyBuffer(wb, rb, nil, make(chan struct{}, 2))
	if wb.String() != "hello, world." {
		t.Errorf("Copy did not work properly")
	}
}

type errBufferWriterCloser struct {
	buf []byte
}

func (cb *errBufferWriterCloser) Write(p []byte) (int, error) {
	return -1, nil
}
func (cb *errBufferWriterCloser) Read(p []byte) (int, error) {
	return 0, nil
}

func (*errBufferWriterCloser) Close() error {
	return nil
}

func TestCopyWriteToErr(t *testing.T) {
	backend := new(copyBufferWriterCloser)
	backend.Write([]byte("123"))
	wb := new(errBufferWriterCloser)
	_, err := http.TestCoyBuffer(wb, backend, nil, make(chan struct{}, 2))
	if err == nil {
		t.Fatalf("error check")
	}
}
