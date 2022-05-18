// Code generated by mockery v2.12.1. DO NOT EDIT.

package mocks

import (
	testing "testing"

	node "github.com/sawantshivaji1997/notionbackup/src/tree/node"
	mock "github.com/stretchr/testify/mock"
)

// Iterator is an autogenerated mock type for the Iterator type
type Iterator struct {
	mock.Mock
}

// Next provides a mock function with given fields:
func (_m *Iterator) Next() (*node.Node, error) {
	ret := _m.Called()

	var r0 *node.Node
	if rf, ok := ret.Get(0).(func() *node.Node); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*node.Node)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewIterator creates a new instance of Iterator. It also registers the testing.TB interface on the mock and a cleanup function to assert the mocks expectations.
func NewIterator(t testing.TB) *Iterator {
	mock := &Iterator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}