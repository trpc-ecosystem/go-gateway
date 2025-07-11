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

package plugin

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-gateway/common/convert"
)

func TestPropsDecoder_Decode(t *testing.T) {
	type user struct {
		Name string `yaml:"name"`
		Age  int    `yaml:"age"`
	}
	u := &user{Name: "quon"}
	decoder := &PropsDecoder{
		Props: u,
	}
	nu := &user{}
	err := decoder.Decode(nu)
	assert.Nil(t, err)
	nu.Age = 19
	assert.EqualValues(t, decoder.DecodedProps, &user{Name: "quon", Age: 19})

	decoder = &PropsDecoder{}
	err = decoder.Decode(&user{})
	assert.Nil(t, err)
	decoder = &PropsDecoder{
		Props: &user{},
	}
	err = decoder.Decode(user{})
	assert.NotNil(t, err)
}

func TestMashal(t *testing.T) {

	props, err := yaml.Marshal(make(map[string]interface{}))
	assert.Nil(t, err)
	t.Log(string(props))
	type User struct {
		Name string
	}
	user := &User{}
	if reflect.ValueOf(&user).Kind() != reflect.Ptr {
		t.Error("invalid")
		return
	}
	err = yaml.Unmarshal(props, &user)

	assert.Nil(t, err)
	t.Log(convert.ToJSONStr(user))
	user.Name = "xx"
	t.Log(convert.ToJSONStr(user))
}
