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

package response_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
	"github.com/valyala/fasthttp"
	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-gateway/common/gwmsg"
	"trpc.group/trpc-go/trpc-gateway/common/http"
	"trpc.group/trpc-go/trpc-gateway/plugin"
	"trpc.group/trpc-go/trpc-gateway/plugin/transformer/response"
	"trpc.group/trpc-go/trpc-go/errs"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"
)

func TestPlugin_CheckConfig(t *testing.T) {
	// Delete response header
	options := &response.Options{
		RemoveHeaders: []*response.KeysConfig{
			{
				Keys: []string{
					"header_to_remover",
				},
				StatusCodes: []int{401},
				TRPCCodes:   []trpcpb.TrpcRetCode{5000},
			},
		},
	}
	p := &response.Plugin{}
	_ = p.Setup("", nil)
	decoder := &plugin.PropsDecoder{Props: options}
	err := p.CheckConfig("", decoder)
	assert.Nil(t, err)

	fctx := &fasthttp.RequestCtx{}
	fctx.Response.Header.Set("header_to_remover", "val")
	fctx.Response.SetStatusCode(401)
	ctx := http.WithRequestContext(context.Background(), fctx)
	ctx, msg := gwmsg.WithNewGWMessage(ctx)
	msg.WithPluginConfig(response.PluginName, decoder.DecodedProps)
	_, err = response.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(fctx.Response.Header.Peek("header_to_remover")))
	options.RemoveHeaders = nil

	options.RemoveJSON = []*response.KeysConfig{
		{
			Keys: []string{
				"common.suid",
			},
			TRPCCodes: []trpcpb.TrpcRetCode{5000},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig(response.PluginName, decoder.DecodedProps)
	fctx.Response.SetBodyString(`{"common":{"suid":"xxx"}}`)
	_, err = response.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, errs.New(5000, "err")
	})
	assert.NotNil(t, err)
	assert.False(t, gjson.GetBytes(fctx.Response.Body(), "common.suid").Exists())
	t.Log(string(fctx.Response.Body()))
	options.RemoveJSON = nil

	// Rename header
	options.RenameHeaders = []*response.KeysConfig{
		{
			Keys: []string{
				"old_header:new_header",
			},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig(response.PluginName, decoder.DecodedProps)
	fctx.Response.SetStatusCode(fasthttp.StatusOK)
	fctx.Response.Header.Set("old_header", "val")
	_, err = response.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(fctx.Response.Header.Peek("old_header")))
	assert.Equal(t, "", string(fctx.Response.Header.Peek("new_header")))
	fctx.Response.Header.DisableNormalizing()
	assert.Equal(t, "val", string(fctx.Response.Header.Peek("new_header")))
	fctx.Response.Header.EnableNormalizing()

	options.RenameHeaders = nil

	// Rename JSON
	options.RenameJSON = []*response.KeysConfig{
		{
			Keys: []string{
				"common.old_json_key:common.new_json_key",
			},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig(response.PluginName, decoder.DecodedProps)
	fctx.Response.SetStatusCode(fasthttp.StatusOK)
	fctx.Response.SetBodyString(`{"common":{"old_json_key":"val"}}`)
	_, err = response.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(fctx.Response.Header.Peek("old_header")))
	assert.Equal(t, "val", gjson.GetBytes(fctx.Response.Body(), "common.new_json_key").Value())
	assert.Nil(t, gjson.GetBytes(fctx.Response.Body(), "common.old_json_key").Value())
	t.Log(string(fctx.Response.Body()))
	options.RenameJSON = nil

	// Add header
	options.AddHeaders = []*response.KeysConfig{
		{
			Keys: []string{
				"header_to_add:val_to_add",
			},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig(response.PluginName, decoder.DecodedProps)
	_, err = response.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "val_to_add", string(fctx.Response.Header.Peek("header_to_add")))
	options.AddHeaders = nil

	// Add JSON key
	options.AddJSON = []*response.KeysConfig{
		{
			Keys: []string{
				"common",
			},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)
	options.AddJSON = []*response.KeysConfig{
		{
			Keys: []string{
				"common:xxx:invalid_type",
			},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)
	options.AddJSON = []*response.KeysConfig{
		{
			Keys: []string{
				"common:xxx:number",
			},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)

	options.AddJSON = []*response.KeysConfig{
		{
			Keys: []string{
				"common:xxx:bool",
			},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)
	options.AddJSON = nil

	options.ReplaceJSON = []*response.KeysConfig{
		{
			Keys: []string{
				"common",
			},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)
	options.ReplaceJSON = nil

	options.AppendJSON = []*response.KeysConfig{
		{
			Keys: []string{
				"common",
			},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)
	options.AppendJSON = nil

	options.RenameJSON = []*response.KeysConfig{
		{
			Keys: []string{
				"common",
			},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)
	options.RenameJSON = nil

	options.RenameHeaders = []*response.KeysConfig{
		{
			Keys: []string{
				"common",
			},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)
	options.RenameHeaders = nil

	options.AddHeaders = []*response.KeysConfig{
		{
			Keys: []string{
				"common",
			},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)
	options.AddHeaders = nil

	options.ReplaceHeaders = []*response.KeysConfig{
		{
			Keys: []string{
				"common",
			},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)
	options.ReplaceHeaders = nil

	options.AppendHeaders = []*response.KeysConfig{
		{
			Keys: []string{
				"common",
			},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.NotNil(t, err)
	options.AppendHeaders = nil

	options.AddJSON = []*response.KeysConfig{
		{
			Keys: []string{
				"common.json_key_to_add:val_to_add:string",
			},
		},
		{
			Keys: []string{
				"common.json_key_to_add2:true:bool",
			},
		},
		{
			Keys: []string{
				"common.json_key_to_add3:true:",
			},
		},
		{
			Keys: []string{
				":val_to_add",
			},
		},
		{
			Keys: []string{
				"",
			},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig(response.PluginName, decoder.DecodedProps)
	_, err = response.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "val_to_add", gjson.GetBytes(fctx.Response.Body(), "common.json_key_to_add").Value())
	options.AddJSON = nil

	options.AddJSON = []*response.KeysConfig{
		{
			Keys: []string{
				"common.json_key_to_add:val_to_add:string",
			},
			KVs: []*response.KV{{
				Key:          "",
				Val:          "",
				ConvertedVal: nil,
			}},
		},
	}
	assert.Nil(t, err)
	msg.WithPluginConfig(response.PluginName, options)
	_, err = response.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	options.AddJSON = nil

	options.RemoveJSON = []*response.KeysConfig{
		{
			Keys: []string{
				"",
			},
			KVs: []*response.KV{{
				Key:          "",
				Val:          "",
				ConvertedVal: nil,
			}},
		},
	}
	assert.Nil(t, err)
	msg.WithPluginConfig(response.PluginName, options)
	_, err = response.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	options.RemoveJSON = nil

	options.AppendJSON = []*response.KeysConfig{
		{
			Keys: []string{
				"",
			},
			KVs: []*response.KV{{
				Key:          "",
				Val:          "",
				ConvertedVal: nil,
			}},
		},
	}
	assert.Nil(t, err)
	msg.WithPluginConfig(response.PluginName, options)
	_, err = response.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	options.AppendJSON = nil

	options.ReplaceJSON = []*response.KeysConfig{
		{
			Keys: []string{
				"",
			},
			KVs: []*response.KV{{
				Key:          "",
				Val:          "",
				ConvertedVal: nil,
			}},
		},
	}
	assert.Nil(t, err)
	msg.WithPluginConfig(response.PluginName, options)
	_, err = response.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	options.ReplaceJSON = nil

	options.RenameJSON = []*response.KeysConfig{
		{
			Keys: []string{
				"",
			},
			KVs: []*response.KV{{
				Key:          "rename_json_key",
				Val:          "",
				ConvertedVal: nil,
			}},
		},
	}
	assert.Nil(t, err)
	fctx.Response.SetBodyString(`{"rename_json_key":""}`)
	msg.WithPluginConfig(response.PluginName, options)
	_, err = response.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	options.RenameJSON = nil

	options.AllowJSON = []*response.KeysConfig{
		{
			Keys: []string{
				"",
			},
			KVs: []*response.KV{{
				Key:          "",
				Val:          "",
				ConvertedVal: nil,
			}},
		},
	}
	assert.Nil(t, err)
	msg.WithPluginConfig(response.PluginName, options)
	_, err = response.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	options.AllowJSON = nil

	// replace header
	options.ReplaceHeaders = []*response.KeysConfig{
		{
			Keys: []string{
				"header_to_replace:new_val",
				"header_to_replace_empty:new_val",
			},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig(response.PluginName, decoder.DecodedProps)
	fctx.Response.Header.Set("header_to_replace", "old_val")
	_, err = response.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "new_val", string(fctx.Response.Header.Peek("header_to_replace")))
	assert.Equal(t, "", string(fctx.Response.Header.Peek("header_to_replace_empty")))
	options.ReplaceHeaders = nil

	// replace json
	options.ReplaceJSON = []*response.KeysConfig{
		{
			Keys: []string{
				"json_to_replace:123:number",
			},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig(response.PluginName, decoder.DecodedProps)
	fctx.Response.SetBodyString(`{"json_to_replace":"old_val"}`)
	_, err = response.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, float64(123), gjson.GetBytes(fctx.Response.Body(), "json_to_replace").Value())
	options.ReplaceJSON = nil

	// append headers
	options.AppendHeaders = []*response.KeysConfig{
		{
			Keys: []string{
				"header_to_append:append_val",
			},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig(response.PluginName, decoder.DecodedProps)
	fctx.Response.Header.Add("header_to_append", "curr_val")
	_, err = response.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "curr_val", string(fctx.Response.Header.PeekAll("header_to_append")[0]))
	assert.Equal(t, "append_val", string(fctx.Response.Header.PeekAll("header_to_append")[1]))
	options.AppendHeaders = nil

	// append json
	options.AppendJSON = []*response.KeysConfig{
		{
			Keys: []string{
				"json_key_to_append_number:45:number",
			},
		},
		{
			Keys: []string{
				"json_key_to_append:true:bool",
			},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)

	msg.WithPluginConfig(response.PluginName, decoder.DecodedProps)
	fctx.Response.SetBodyString(`{"json_key_to_append_number":"aa"}`)
	_, err = response.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.ElementsMatch(t, []interface{}{"aa", float64(45)}, gjson.GetBytes(fctx.Response.Body(),
		"json_key_to_append_number").Value())
	assert.Equal(t, true, gjson.GetBytes(fctx.Response.Body(), "json_key_to_append").Value())
	options.AppendJSON = nil

	// replace body
	options.ReplaceBody = []*response.KeysConfig{
		{
			Keys: []string{
				`{"code":401,"msg":"success"}`,
			},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig(response.PluginName, decoder.DecodedProps)
	fctx.Response.SetBodyString(`{"json_key_to_append_number":"aa"}`)
	_, err = response.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, `{"code":401,"msg":"success"}`, string(fctx.Response.Body()))
	options.ReplaceBody = nil

	// allow json
	options.AllowJSON = []*response.KeysConfig{
		{
			Keys: []string{
				"code",
				"msg",
				"data.suid",
			},
		},
	}
	err = p.CheckConfig("", decoder)
	assert.Nil(t, err)
	msg.WithPluginConfig(response.PluginName, decoder.DecodedProps)
	fctx.Response.SetBodyString(`{"code":4001,"msg":"success","data":{"suid":"xxx"},"rsp_code":444}`)
	_, err = response.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, `{"code":4001,"msg":"success","data":{"suid":"xxx"}}`, string(fctx.Response.Body()))
	options.AllowJSON = nil

	// Conversion failed
	msg.WithPluginConfig(response.PluginName, nil)
	fctx.Response.SetBodyString(`{"code":4001,"msg":"success","data":{"suid":"xxx"},"rsp_code":444}`)
	_, err = response.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	// Conversion failed
	fctx.Response.SetBodyString(`{"code":4001,"msg":"success","data":{"suid":"xxx"},"rsp_code":444}`)
	_, err = response.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
	assert.Nil(t, err)

	ctx, msg = gwmsg.WithNewGWMessage(context.Background())
	msg.WithPluginConfig(response.PluginName, decoder.DecodedProps)
	_, _ = response.ServerFilter(ctx, nil, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, nil
	})
}

func TestCheckConfig(t *testing.T) {

	options := &response.Options{
		RemoveHeaders: []*response.KeysConfig{
			{
				Keys: []string{
					"header_to_remover",
				},
				StatusCodes: []int{401},
				TRPCCodes:   []trpcpb.TrpcRetCode{5000},
			},
		},
		RemoveJSON: []*response.KeysConfig{
			{
				Keys: []string{
					"header_to_remover",
				},
				StatusCodes: []int{401},
				TRPCCodes:   []trpcpb.TrpcRetCode{5000},
			},
		},
		RenameHeaders: []*response.KeysConfig{
			{
				Keys: []string{
					"k:v",
				},
				StatusCodes: []int{401},
				TRPCCodes:   []trpcpb.TrpcRetCode{5000},
			},
		},
		RenameJSON: []*response.KeysConfig{
			{
				Keys: []string{
					"k:v:string",
				},
				StatusCodes: []int{401},
				TRPCCodes:   []trpcpb.TrpcRetCode{5000},
			},
		},
		AddHeaders: []*response.KeysConfig{
			{
				Keys: []string{
					"k:v",
				},
				StatusCodes: []int{401},
				TRPCCodes:   []trpcpb.TrpcRetCode{5000},
			},
		},
		AddJSON: []*response.KeysConfig{
			{
				Keys: []string{
					"k:true:bool",
				},
				StatusCodes: []int{401},
				TRPCCodes:   []trpcpb.TrpcRetCode{5000},
			},
		},
		ReplaceHeaders: []*response.KeysConfig{
			{
				Keys: []string{
					"k:v",
				},
				StatusCodes: []int{401},
				TRPCCodes:   []trpcpb.TrpcRetCode{5000},
			},
		},
		ReplaceJSON: []*response.KeysConfig{
			{
				Keys: []string{
					"k:123:number",
				},
				StatusCodes: []int{401},
				TRPCCodes:   []trpcpb.TrpcRetCode{5000},
			},
		},
		AppendHeaders: []*response.KeysConfig{
			{
				Keys: []string{
					"k:v",
				},
				StatusCodes: []int{401},
				TRPCCodes:   []trpcpb.TrpcRetCode{5000},
			},
		},
		AppendJSON: []*response.KeysConfig{
			{
				Keys: []string{
					"k:v:string",
				},
				StatusCodes: []int{401},
				TRPCCodes:   []trpcpb.TrpcRetCode{5000},
			},
		},
		ReplaceBody: []*response.KeysConfig{
			{
				Keys: []string{
					`{"code":401}`,
				},
				StatusCodes: []int{401},
				TRPCCodes:   []trpcpb.TrpcRetCode{5000},
			},
		},
		AllowJSON: []*response.KeysConfig{
			{
				Keys: []string{
					"code",
					"msg",
					"data",
				},
				StatusCodes: []int{401},
				TRPCCodes:   []trpcpb.TrpcRetCode{5000},
			},
		},
	}
	p := &response.Plugin{}
	decoder := &plugin.PropsDecoder{Props: options}
	err := p.CheckConfig("", decoder)
	assert.Nil(t, err)
	c, err := yaml.Marshal(options)
	assert.Nil(t, err)
	fmt.Println(string(c))
}
