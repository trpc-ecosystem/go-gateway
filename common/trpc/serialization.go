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

package trpc

import (
	"mime"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
)

// contentTypeSerializationTypeMap is the mapping between HTTP content types and tRPC serialization types,
// used for protocol pass through.
var contentTypeSerializationTypeMap = map[string]interface{}{
	"application/json":                  codec.SerializationTypeJSON,
	"application/protobuf":              codec.SerializationTypePB,
	"application/x-protobuf":            codec.SerializationTypePB,
	"application/pb":                    codec.SerializationTypePB,
	"application/proto":                 codec.SerializationTypePB,
	"application/flatbuffer":            codec.SerializationTypeFlatBuffer,
	"application/octet-stream":          codec.SerializationTypeNoop,
	"application/x-www-form-urlencoded": codec.SerializationTypeForm,
	"application/xml":                   codec.SerializationTypeXML,
	"text/xml":                          codec.SerializationTypeTextXML,
	"multipart/form-data":               codec.SerializationTypeFormData,
}

// Register registers the mapping relationship between custom http content type and trpc serialization type
func Register(contentType string, serializationType int) error {
	// when the trpc-go service is started, the registration will not be loaded concurrently and no locking is required.
	contentTypeSerializationTypeMap[contentType] = serializationType
	return nil
}

// GetSerializationType gets trpc serialization type through http content type.
func GetSerializationType(contentType string) (int, error) {
	baseCT, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return 0, errs.Wrapf(err, codec.SerializationTypeUnsupported, "invalid content type:%s", contentType)
	}

	st, ok := contentTypeSerializationTypeMap[baseCT]
	if !ok {
		return 0, errs.Newf(codec.SerializationTypeUnsupported, "unsupported content type:%s", contentType)
	}
	return st.(int), nil
}
