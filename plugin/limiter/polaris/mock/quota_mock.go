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

// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/polarismesh/polaris-go/api (interfaces: QuotaFuture)

// Package mock_api is a generated GoMock package.
package mock_api

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	model "github.com/polarismesh/polaris-go/pkg/model"
)

// MockQuotaFuture is a mock of QuotaFuture interface.
type MockQuotaFuture struct {
	ctrl     *gomock.Controller
	recorder *MockQuotaFutureMockRecorder
}

// MockQuotaFutureMockRecorder is the mock recorder for MockQuotaFuture.
type MockQuotaFutureMockRecorder struct {
	mock *MockQuotaFuture
}

// NewMockQuotaFuture creates a new mock instance.
func NewMockQuotaFuture(ctrl *gomock.Controller) *MockQuotaFuture {
	mock := &MockQuotaFuture{ctrl: ctrl}
	mock.recorder = &MockQuotaFutureMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockQuotaFuture) EXPECT() *MockQuotaFutureMockRecorder {
	return m.recorder
}

// Done mocks base method.
func (m *MockQuotaFuture) Done() <-chan struct{} {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Done")
	ret0, _ := ret[0].(<-chan struct{})
	return ret0
}

// Done indicates an expected call of Done.
func (mr *MockQuotaFutureMockRecorder) Done() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Done", reflect.TypeOf((*MockQuotaFuture)(nil).Done))
}

// Get mocks base method.
func (m *MockQuotaFuture) Get() *model.QuotaResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get")
	ret0, _ := ret[0].(*model.QuotaResponse)
	return ret0
}

// Get indicates an expected call of Get.
func (mr *MockQuotaFutureMockRecorder) Get() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockQuotaFuture)(nil).Get))
}

// GetImmediately mocks base method.
func (m *MockQuotaFuture) GetImmediately() *model.QuotaResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetImmediately")
	ret0, _ := ret[0].(*model.QuotaResponse)
	return ret0
}

// GetImmediately indicates an expected call of GetImmediately.
func (mr *MockQuotaFutureMockRecorder) GetImmediately() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetImmediately", reflect.TypeOf((*MockQuotaFuture)(nil).GetImmediately))
}

// Release mocks base method.
func (m *MockQuotaFuture) Release() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Release")
}

// Release indicates an expected call of Release.
func (mr *MockQuotaFutureMockRecorder) Release() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Release", reflect.TypeOf((*MockQuotaFuture)(nil).Release))
}
