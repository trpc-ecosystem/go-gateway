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

package request

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/plugin"
)

func TestPlugin_CheckConfig(t *testing.T) {
	options := &Options{
		RewriteHost:    "r.inews.qq.com",
		RemoveHeaders:  []string{"remove_header"},
		RemoveQueryStr: []string{"remove_query_str"},
		RemoveBody:     []string{"remove_query_body"},
		RenameHeaders:  []string{"", ":aaa", "old_header:new_header"},
		RenameQueryStr: []string{"", ":aaa", "old_query_str:new_query_str"},
		RenameBody:     []string{"", ":aaa", "old_query_body:new_query_body"},
		AddHeaders:     []string{"", ":aaa", "new_header:new_header_val"},
		AddQueryStr:    []string{"", ":aaa", "new_query_str:new_query_val"},
		AddBody: []string{"", ":aaa",
			"new_query_body:new_query_body_val",
			"key_bool:true:bool",
			"key_num:1:number",
			"key_float_num:1.0:number",
			"key_str:hi:string",
		},
	}
	p := &Plugin{}
	_ = p.Setup("", nil)
	decoder := &plugin.PropsDecoder{
		Props: options,
	}
	err := p.CheckConfig("", decoder)
	assert.Nil(t, err)
	// RenameHeaders invalid
	tmp := options.RenameHeaders
	options.RenameHeaders = []string{"", "old_header"}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)
	options.RenameHeaders = tmp

	// RenameQueryStr invalid
	tmp = options.RenameQueryStr
	options.RenameQueryStr = []string{"", "old_header", ""}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)
	options.RenameQueryStr = tmp

	// RenameQueryStr invalid
	tmp = options.RenameBody
	options.RenameBody = []string{"", "old_header"}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)
	options.RenameBody = tmp

	// AddHeaders invalid
	tmp = options.AddHeaders
	options.AddHeaders = []string{"", "old_header"}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)
	options.AddHeaders = tmp

	// AddQueryStr invalid
	tmp = options.AddQueryStr
	options.AddQueryStr = []string{"", "old_header"}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)
	options.AddQueryStr = tmp

	// AddQueryBody invalid
	tmp = options.AddBody
	options.AddBody = []string{":key", "old_header", ""}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)
	options.AddBody = tmp

	// AddBody invalid
	tmp = options.AddBody
	options.AddBody = []string{
		"invalid_type_oldkey:invalid_type_newkey:a:number",
	}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)
	options.AddBody = []string{
		"not_support_oldkey:not_support_newkey:1:int32",
	}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)
	options.AddBody = tmp
}

func TestServerFilter(t *testing.T) {
	options := &Options{
		RewriteHost:    "r.inews.qq.com",
		RemoveHeaders:  []string{"remove_header"},
		RemoveQueryStr: []string{"remove_query_str"},
		RemoveBody:     []string{"remove_query_body"},
		RenameHeaders:  []string{"old_header:new_header"},
		RenameQueryStr: []string{"old_query_str:new_query_str_key"},
		RenameBody:     []string{"old_query_body:new_query_body_key"},
		AddHeaders:     []string{"new_header_key:new_header_val"},
		AddQueryStr:    []string{"new_query_str:new_query_val"},
		AddBody:        []string{"new_query_body:new_query_body_val"},
	}
	p := &Plugin{}
	decoder := &plugin.PropsDecoder{
		Props: options,
	}
	err := p.CheckConfig("", decoder)
	assert.Nil(t, err)
	// Failed to retrieve plugin configuration
	_, err = ServerFilter(context.Background(), nil, func(ctx context.Context,
		req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.NotNil(t, err)

	ctx, msg := gwmsg.WithNewGWMessage(context.Background())
	msg.WithPluginConfig(pluginName, "invalid option")
	// Plugin configuration assertion failed
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.NotNil(t, err)
	msg.WithPluginConfig(pluginName, decoder.DecodedProps)
	// Failed to retrieve fctx
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	fctx := &fasthttp.RequestCtx{}
	fctx.Request.Header.SetContentTypeBytes(strPostArgsContentType)
	fctx.Request.SetHost("shizi.qq.com")
	fctx.Request.Header.Set("remove_header", "val")
	fctx.Request.URI().QueryArgs().Set("remove_query_str", "val")
	fctx.Request.Header.Set("old_header", "val")
	fctx.Request.URI().QueryArgs().Set("old_query_str", "val")
	fctx.Request.PostArgs().Set("old_query_body", "val")
	ctx = http.WithRequestContext(ctx, fctx)
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	assert.Equal(t, "r.inews.qq.com", string(fctx.Host()))
	assert.Equal(t, "", string(fctx.Request.Header.Peek("remove_header")))
	assert.Equal(t, "", string(fctx.Request.URI().QueryArgs().Peek("remove_query_str")))
	assert.Equal(t, "", string(fctx.Request.Header.Peek("old_header")))
	assert.Equal(t, "", string(fctx.Request.Header.Peek("new_header")))
	fctx.Request.Header.DisableNormalizing()
	assert.Equal(t, "val", string(fctx.Request.Header.Peek("new_header")))
	fctx.Request.Header.EnableNormalizing()
	assert.Equal(t, "", string(fctx.Request.URI().QueryArgs().Peek("old_query_str")))
	assert.Equal(t, "val", string(fctx.Request.URI().QueryArgs().Peek("new_query_str_key")))
	assert.Equal(t, "", string(fctx.Request.PostArgs().Peek("old_query_body")))
	assert.Equal(t, "val", string(fctx.Request.PostArgs().Peek("new_query_body_key")))
	assert.Equal(t, "new_header_val", string(fctx.Request.Header.Peek("new_header_key")))
	assert.Equal(t, "new_query_val", string(fctx.Request.URI().QueryArgs().Peek("new_query_str")))
	assert.Equal(t, "new_query_body_val", string(fctx.Request.PostArgs().Peek("new_query_body")))
}

func TestReserv(t *testing.T) {
	options := &Options{
		RewriteHost:     "r.inews.qq.com",
		ReserveHeaders:  []string{"reserve_header", "host", "content-type"},
		RemoveHeaders:   []string{"reserve_header"},
		ReserveQueryStr: []string{"reserve_query_str"},
		RemoveQueryStr:  []string{"reserve_query_str"},
		ReserveBody:     []string{"reserve_query_body"},
		RemoveBody:      []string{"reserve_query_body"},
	}
	p := &Plugin{}
	decoder := &plugin.PropsDecoder{
		Props: options,
	}
	err := p.CheckConfig("", decoder)
	assert.Nil(t, err)
	// Failed to retrieve plugin configuration
	_, err = ServerFilter(context.Background(), nil, func(ctx context.Context,
		req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.NotNil(t, err)

	ctx, msg := gwmsg.WithNewGWMessage(context.Background())
	msg.WithPluginConfig(pluginName, "invalid option")
	// Plugin configuration assertion failed
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.NotNil(t, err)
	msg.WithPluginConfig(pluginName, decoder.DecodedProps)
	// Failed to retrieve fctx
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	fctx := &fasthttp.RequestCtx{}
	fctx.Request.SetHost("shizi.qq.com")
	fctx.Request.Header.SetContentTypeBytes(strPostArgsContentType)
	fctx.Request.Header.Set("reserve_header", "val")
	fctx.Request.URI().QueryArgs().Set("reserve_query_str", "val")
	fctx.Request.PostArgs().Set("reserve_query_body", "val")
	ctx = http.WithRequestContext(ctx, fctx)
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	assert.Equal(t, "r.inews.qq.com", string(fctx.Host()))
	assert.Equal(t, "val", string(fctx.Request.Header.Peek("reserve_header")))
	assert.Equal(t, "val", string(fctx.Request.URI().QueryArgs().Peek("reserve_query_str")))
	assert.Equal(t, "val", string(fctx.Request.PostArgs().Peek("reserve_query_body")))

	options = &Options{
		RewriteHost:     "r.inews.qq.com",
		ReserveHeaders:  []string{"content-type"},
		RemoveHeaders:   []string{"reserve_header"},
		ReserveQueryStr: []string{"-1"},
		RemoveQueryStr:  []string{"reserve_query_str"},
		ReserveBody:     []string{"-1"},
		RemoveBody:      []string{"reserve_query_body"},
	}
	decoder = &plugin.PropsDecoder{
		Props: options,
	}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)

	fctx = &fasthttp.RequestCtx{}
	fctx.Request.Header.SetContentTypeBytes(strPostArgsContentType)
	fctx.Request.SetHost("shizi.qq.com")
	fctx.Request.Header.Set("reserve_header", "val")
	fctx.Request.URI().QueryArgs().Set("reserve_query_str", "val")
	fctx.Request.PostArgs().Set("reserve_query_body", "val")
	msg.WithPluginConfig(pluginName, decoder.DecodedProps)

	ctx = http.WithRequestContext(ctx, fctx)
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "r.inews.qq.com", string(fctx.Host()))
	assert.Equal(t, "", string(fctx.Request.Header.Peek("reserve_header")))
	assert.Equal(t, "", string(fctx.Request.URI().QueryArgs().Peek("reserve_query_str")))
	assert.Equal(t, "", string(fctx.Request.PostArgs().Peek("reserve_query_body")))
}

func TestBodyOptions(t *testing.T) {
	// Preserve query body
	options := &Options{
		ReserveBody: []string{"reserve_query_body"},
	}
	p := &Plugin{}
	decoder := &plugin.PropsDecoder{
		Props: options,
	}
	err := p.CheckConfig("", decoder)
	assert.Nil(t, err)

	ctx, msg := gwmsg.WithNewGWMessage(context.Background())
	msg.WithPluginConfig(pluginName, decoder.DecodedProps)
	fctx := &fasthttp.RequestCtx{}
	fctx.Request.Header.SetContentTypeBytes(strPostArgsContentType)
	fctx.Request.PostArgs().Set("reserve_query_body", "val")
	ctx = http.WithRequestContext(ctx, fctx)
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "val", string(fctx.Request.PostArgs().Peek("reserve_query_body")))

	// json body
	fctx.Request.SetBodyString(`{"reserve_query_body":"val","another":"anther_val"}`)
	fctx.Request.Header.SetContentTypeBytes(strJSONContentType)
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, `{"reserve_query_body":"val"}`, string(fctx.Request.Body()))

	// multipart form body
	fctx.Request.Header.SetContentType("multipart/form-data; boundary=----WebKitFormBoundaryWrGPeYfXRBUkQimG")
	formBody := `------WebKitFormBoundaryWrGPeYfXRBUkQimG
Content-Disposition: form-data; name="reserve_query_body"

val
------WebKitFormBoundaryWrGPeYfXRBUkQimG
Content-Disposition: form-data; name="another"

another_val
------WebKitFormBoundaryWrGPeYfXRBUkQimG--`
	fctx.Request.SetBodyString(formBody)
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	// Cannot preserve operations for multiform type requests, as handling file fields is more complex
	assert.Equal(t, formBody, string(fctx.Request.Body()))

	// Delete all body parameters
	options.ReserveBody = []string{"-1"}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig(pluginName, decoder.DecodedProps)
	fctx.Request.Header.SetContentTypeBytes(strPostArgsContentType)
	fctx.Request.PostArgs().Set("reserve_query_body", "val")
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "", string(fctx.Request.PostArgs().Peek("reserve_query_body")))

	// Delete all JSON body parameters
	options.ReserveBody = []string{"-1"}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig(pluginName, decoder.DecodedProps)
	fctx.Request.Header.SetContentTypeBytes(strJSONContentType)
	fctx.Request.SetBodyString(`{"reserve_query_body":"val"}`)
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "", string(fctx.Request.PostArgs().Peek("reserve_query_body")))

	// Delete all multipart form body parameters
	options.ReserveBody = []string{"-1"}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig(pluginName, decoder.DecodedProps)
	fctx.Request.Header.SetContentType("multipart/form-data; boundary=----WebKitFormBoundaryWrGPeYfXRBUkQimG")
	fctx.Request.SetBodyString(formBody)
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, formBody, string(fctx.Request.Body()))

	// Delete specified query body parameters
	options.ReserveBody = []string{}
	options.RemoveBody = []string{"reserve_query_body"}

	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig(pluginName, decoder.DecodedProps)
	fctx.Request.Header.SetContentTypeBytes(strPostArgsContentType)
	fctx.Request.PostArgs().Set("reserve_query_body", "val")
	fctx.Request.PostArgs().Set("another", "another_val")
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "", string(fctx.Request.PostArgs().Peek("reserve_query_body")))
	assert.Equal(t, "another_val", string(fctx.Request.PostArgs().Peek("another")))

	// Delete specified JSON body parameters
	fctx.Request.Header.SetContentTypeBytes(strJSONContentType)
	fctx.Request.SetBodyString(`{"reserve_query_body":"val","another":"anther_val"}`)
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, `{"another":"anther_val"}`, string(fctx.Request.Body()))

	// Delete specified multipart form body parameters
	fctx.Request.SetBodyString(formBody)
	fctx.Request.Header.SetContentType("multipart/form-data; boundary=----WebKitFormBoundaryWrGPeYfXRBUkQimG")
	_, err = ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	form, err := fctx.MultipartForm()
	assert.Nil(t, err)

	assert.Equal(t, 0, len(form.Value["reserve_query_body"]))
	assert.Equal(t, "another_val", form.Value["another"][0])
	assert.Equal(t, "------WebKitFormBoundaryWrGPeYfXRBUkQimG\r\nContent-Disposition: form-data; "+
		"name=\"another\"\r\n\r\nanother_val\r\n------WebKitFormBoundaryWrGPeYfXRBUkQimG--\r\n",
		string(fctx.Request.Body()))
}

func Test_addBody(t *testing.T) {
	fctx := &fasthttp.RequestCtx{}
	fctx.Request.Header.SetContentTypeBytes(strPostArgsContentType)
	addBody(context.Background(), &fctx.Request, &KV{
		Key:          "key",
		Val:          "val",
		ConvertedVal: "val",
	})
	assert.Equal(t, "val", string(fctx.Request.PostArgs().Peek("key")))
	addBody(context.Background(), &fctx.Request, &KV{
		Key:          "key_num",
		Val:          "1",
		ConvertedVal: 1,
	})
	assert.Equal(t, "1", string(fctx.Request.PostArgs().Peek("key_num")))
	addBody(context.Background(), &fctx.Request, &KV{
		Key:          "key_bool",
		Val:          "true",
		ConvertedVal: true,
	})
	assert.Equal(t, "true", string(fctx.Request.PostArgs().Peek("key_bool")))
	fctx.Request.ResetBody()
	fctx.Request.Header.SetContentTypeBytes(strJSONContentType)
	fctx.Request.SetBodyString(`{}`)
	addBody(context.Background(), &fctx.Request, &KV{
		Key:          "key",
		Val:          "val",
		ConvertedVal: "val",
	})
	assert.Equal(t, `{"key":"val"}`, string(fctx.Request.Body()))
	fctx.Request.ResetBody()
	addBody(context.Background(), &fctx.Request, &KV{
		Key:          "key_num",
		Val:          "1",
		ConvertedVal: 1,
	})
	assert.Equal(t, `{"key_num":1}`, string(fctx.Request.Body()))
	fctx.Request.ResetBody()
	addBody(context.Background(), &fctx.Request, &KV{
		Key:          "key_num",
		Val:          "1.0",
		ConvertedVal: 1.0,
	})
	assert.Equal(t, `{"key_num":1}`, string(fctx.Request.Body()))
	fctx.Request.ResetBody()
	addBody(context.Background(), &fctx.Request, &KV{
		Key:          "key_bool",
		Val:          "true",
		ConvertedVal: true,
	})
	assert.Equal(t, `{"key_bool":true}`, string(fctx.Request.Body()))
	fctx.Request.ResetBody()

	formBody := `------WebKitFormBoundaryWrGPeYfXRBUkQimG
Content-Disposition: form-data; name="another"

another_val
------WebKitFormBoundaryWrGPeYfXRBUkQimG--`
	fctx.Request.SetBodyString(formBody)
	fctx.Request.Header.SetContentType("multipart/form-data; boundary=----WebKitFormBoundaryWrGPeYfXRBUkQimG")
	addBody(context.Background(), &fctx.Request, &KV{
		Key:          "key",
		Val:          "val",
		ConvertedVal: "val",
	})

	form, err := fctx.Request.MultipartForm()
	assert.Nil(t, err)
	assert.Equal(t, "val", form.Value["key"][0])
	addBody(context.Background(), &fctx.Request, &KV{
		Key:          "key_num",
		Val:          "1",
		ConvertedVal: 1,
	})
	assert.Equal(t, "1", form.Value["key_num"][0])
	addBody(context.Background(), &fctx.Request, &KV{
		Key:          "key_bool",
		Val:          "true",
		ConvertedVal: true,
	})
	assert.Equal(t, "true", form.Value["key_bool"][0])

	fctx.Request.Header.SetContentType("application/plain")
	fctx.Request.SetBodyString("")
	addBody(context.Background(), &fctx.Request, &KV{
		Key:          "key",
		Val:          "val",
		ConvertedVal: "key1",
	})
	assert.Equal(t, ``, string(fctx.Request.Body()))
}

func Test_renameBody(t *testing.T) {
	fctx := &fasthttp.RequestCtx{}
	fctx.Request.Header.SetContentTypeBytes(strPostArgsContentType)
	fctx.Request.PostArgs().Set("key", "val")
	modified := renameBody(context.Background(), &fctx.Request, &KV{
		Key: "key",
		Val: "key1",
	})
	assert.True(t, modified)
	assert.Equal(t, "val", string(fctx.Request.PostArgs().Peek("key1")))
	// 为找到要修改参数时，忽略变更，获取参数时应为空值
	modified = renameBody(context.Background(), &fctx.Request, &KV{
		Key: "key2",
		Val: "key2",
	})
	assert.False(t, modified)
	assert.Equal(t, "", string(fctx.Request.PostArgs().Peek("key2")))

	fctx.Request.Header.SetContentTypeBytes(strJSONContentType)
	fctx.Request.SetBodyString(`{"key":"val"}`)
	modified = renameBody(context.Background(), &fctx.Request, &KV{
		Key: "key",
		Val: "key1",
	})
	assert.True(t, modified)
	assert.Equal(t, `{"key1":"val"}`, string(fctx.Request.Body()))
	fctx.Request.ResetBody()
	// 为找到要修改参数时，忽略变更
	fctx.Request.Header.SetContentTypeBytes(strJSONContentType)
	fctx.Request.SetBodyString(`{"key":"val"}`)
	modified = renameBody(context.Background(), &fctx.Request, &KV{
		Key: "key1",
		Val: "key1",
	})
	assert.False(t, modified)
	assert.Equal(t, `{"key":"val"}`, string(fctx.Request.Body()))
	fctx.Request.ResetBody()

	formBody := `------WebKitFormBoundaryWrGPeYfXRBUkQimG
Content-Disposition: form-data; name="key"

val
------WebKitFormBoundaryWrGPeYfXRBUkQimG--`
	fctx.Request.SetBodyString(formBody)
	fctx.Request.Header.SetContentType("multipart/form-data; boundary=----WebKitFormBoundaryWrGPeYfXRBUkQimG")
	modified = renameBody(context.Background(), &fctx.Request, &KV{
		Key: "key",
		Val: "key1",
	})

	form, err := fctx.Request.MultipartForm()
	assert.True(t, modified)
	assert.Nil(t, err)
	assert.Equal(t, "val", form.Value["key1"][0])

	modified = renameBody(context.Background(), &fctx.Request, &KV{
		Key: "key2",
		Val: "key2",
	})
	assert.False(t, modified)
	assert.Nil(t, err)

	fctx.Request.Header.SetContentType("application/plain")
	fctx.Request.SetBodyString("")
	renameBody(context.Background(), &fctx.Request, &KV{
		Key: "key",
		Val: "key1",
	})
	assert.Equal(t, ``, string(fctx.Request.Body()))
}

func Test_delBody(t *testing.T) {
	fctx := &fasthttp.RequestCtx{}
	fctx.Request.Header.SetContentType("application/plain")
	fctx.Request.SetBodyString("")
	delBody(context.Background(), &fctx.Request, &Options{
		RemoveBody: []string{},
	})
	assert.Equal(t, ``, string(fctx.Request.Body()))
}
