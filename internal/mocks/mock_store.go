// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/gostuding/goMarket/internal/server (interfaces: Storage)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockStorage is a mock of Storage interface.
type MockStorage struct {
	ctrl     *gomock.Controller
	recorder *MockStorageMockRecorder
}

// MockStorageMockRecorder is the mock recorder for MockStorage.
type MockStorageMockRecorder struct {
	mock *MockStorage
}

// NewMockStorage creates a new mock instance.
func NewMockStorage(ctrl *gomock.Controller) *MockStorage {
	mock := &MockStorage{ctrl: ctrl}
	mock.recorder = &MockStorageMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStorage) EXPECT() *MockStorageMockRecorder {
	return m.recorder
}

// AddOrder mocks base method.
func (m *MockStorage) AddOrder(arg0 context.Context, arg1 int, arg2 string) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddOrder", arg0, arg1, arg2)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AddOrder indicates an expected call of AddOrder.
func (mr *MockStorageMockRecorder) AddOrder(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddOrder", reflect.TypeOf((*MockStorage)(nil).AddOrder), arg0, arg1, arg2)
}

// AddWithdraw mocks base method.
func (m *MockStorage) AddWithdraw(arg0 context.Context, arg1 int, arg2 string, arg3 float32) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddWithdraw", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AddWithdraw indicates an expected call of AddWithdraw.
func (mr *MockStorageMockRecorder) AddWithdraw(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddWithdraw", reflect.TypeOf((*MockStorage)(nil).AddWithdraw), arg0, arg1, arg2, arg3)
}

// Close mocks base method.
func (m *MockStorage) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockStorageMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockStorage)(nil).Close))
}

// GetAccrualOrders mocks base method.
func (m *MockStorage) GetAccrualOrders() []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAccrualOrders")
	ret0, _ := ret[0].([]string)
	return ret0
}

// GetAccrualOrders indicates an expected call of GetAccrualOrders.
func (mr *MockStorageMockRecorder) GetAccrualOrders() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAccrualOrders", reflect.TypeOf((*MockStorage)(nil).GetAccrualOrders))
}

// GetOrders mocks base method.
func (m *MockStorage) GetOrders(arg0 context.Context, arg1 int) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOrders", arg0, arg1)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOrders indicates an expected call of GetOrders.
func (mr *MockStorageMockRecorder) GetOrders(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOrders", reflect.TypeOf((*MockStorage)(nil).GetOrders), arg0, arg1)
}

// GetUserBalance mocks base method.
func (m *MockStorage) GetUserBalance(arg0 context.Context, arg1 int) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserBalance", arg0, arg1)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserBalance indicates an expected call of GetUserBalance.
func (mr *MockStorageMockRecorder) GetUserBalance(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserBalance", reflect.TypeOf((*MockStorage)(nil).GetUserBalance), arg0, arg1)
}

// GetWithdraws mocks base method.
func (m *MockStorage) GetWithdraws(arg0 context.Context, arg1 int) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWithdraws", arg0, arg1)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetWithdraws indicates an expected call of GetWithdraws.
func (mr *MockStorageMockRecorder) GetWithdraws(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWithdraws", reflect.TypeOf((*MockStorage)(nil).GetWithdraws), arg0, arg1)
}

// IsUniqueViolation mocks base method.
func (m *MockStorage) IsUniqueViolation(arg0 error) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsUniqueViolation", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsUniqueViolation indicates an expected call of IsUniqueViolation.
func (mr *MockStorageMockRecorder) IsUniqueViolation(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsUniqueViolation", reflect.TypeOf((*MockStorage)(nil).IsUniqueViolation), arg0)
}

// Login mocks base method.
func (m *MockStorage) Login(arg0 context.Context, arg1, arg2, arg3, arg4 string) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Login", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Login indicates an expected call of Login.
func (mr *MockStorageMockRecorder) Login(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Login", reflect.TypeOf((*MockStorage)(nil).Login), arg0, arg1, arg2, arg3, arg4)
}

// Registration mocks base method.
func (m *MockStorage) Registration(arg0 context.Context, arg1, arg2, arg3, arg4 string) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Registration", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Registration indicates an expected call of Registration.
func (mr *MockStorageMockRecorder) Registration(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Registration", reflect.TypeOf((*MockStorage)(nil).Registration), arg0, arg1, arg2, arg3, arg4)
}

// SetOrderData mocks base method.
func (m *MockStorage) SetOrderData(arg0, arg1 string, arg2 float32) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetOrderData", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetOrderData indicates an expected call of SetOrderData.
func (mr *MockStorageMockRecorder) SetOrderData(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetOrderData", reflect.TypeOf((*MockStorage)(nil).SetOrderData), arg0, arg1, arg2)
}
