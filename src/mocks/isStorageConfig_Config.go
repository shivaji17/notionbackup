// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// isStorageConfig_Config is an autogenerated mock type for the isStorageConfig_Config type
type isStorageConfig_Config struct {
	mock.Mock
}

// isStorageConfig_Config provides a mock function with given fields:
func (_m *isStorageConfig_Config) isStorageConfig_Config() {
	_m.Called()
}

type mockConstructorTestingTnewIsStorageConfig_Config interface {
	mock.TestingT
	Cleanup(func())
}

// newIsStorageConfig_Config creates a new instance of isStorageConfig_Config. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newIsStorageConfig_Config(t mockConstructorTestingTnewIsStorageConfig_Config) *isStorageConfig_Config {
	mock := &isStorageConfig_Config{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
