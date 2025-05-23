// Code generated by mockery. DO NOT EDIT.

//go:build !compile

package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	mock "github.com/stretchr/testify/mock"
)

// MocktgClient is an autogenerated mock type for the tgClient type
type MocktgClient struct {
	mock.Mock
}

type MocktgClient_Expecter struct {
	mock *mock.Mock
}

func (_m *MocktgClient) EXPECT() *MocktgClient_Expecter {
	return &MocktgClient_Expecter{mock: &_m.Mock}
}

// GetUpdatesChan provides a mock function with given fields: config
func (_m *MocktgClient) GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	ret := _m.Called(config)

	if len(ret) == 0 {
		panic("no return value specified for GetUpdatesChan")
	}

	var r0 tgbotapi.UpdatesChannel
	if rf, ok := ret.Get(0).(func(tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel); ok {
		r0 = rf(config)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(tgbotapi.UpdatesChannel)
		}
	}

	return r0
}

// MocktgClient_GetUpdatesChan_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetUpdatesChan'
type MocktgClient_GetUpdatesChan_Call struct {
	*mock.Call
}

// GetUpdatesChan is a helper method to define mock.On call
//   - config tgbotapi.UpdateConfig
func (_e *MocktgClient_Expecter) GetUpdatesChan(config interface{}) *MocktgClient_GetUpdatesChan_Call {
	return &MocktgClient_GetUpdatesChan_Call{Call: _e.mock.On("GetUpdatesChan", config)}
}

func (_c *MocktgClient_GetUpdatesChan_Call) Run(run func(config tgbotapi.UpdateConfig)) *MocktgClient_GetUpdatesChan_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(tgbotapi.UpdateConfig))
	})
	return _c
}

func (_c *MocktgClient_GetUpdatesChan_Call) Return(_a0 tgbotapi.UpdatesChannel) *MocktgClient_GetUpdatesChan_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MocktgClient_GetUpdatesChan_Call) RunAndReturn(run func(tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel) *MocktgClient_GetUpdatesChan_Call {
	_c.Call.Return(run)
	return _c
}

// Send provides a mock function with given fields: c
func (_m *MocktgClient) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	ret := _m.Called(c)

	if len(ret) == 0 {
		panic("no return value specified for Send")
	}

	var r0 tgbotapi.Message
	var r1 error
	if rf, ok := ret.Get(0).(func(tgbotapi.Chattable) (tgbotapi.Message, error)); ok {
		return rf(c)
	}
	if rf, ok := ret.Get(0).(func(tgbotapi.Chattable) tgbotapi.Message); ok {
		r0 = rf(c)
	} else {
		r0 = ret.Get(0).(tgbotapi.Message)
	}

	if rf, ok := ret.Get(1).(func(tgbotapi.Chattable) error); ok {
		r1 = rf(c)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MocktgClient_Send_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Send'
type MocktgClient_Send_Call struct {
	*mock.Call
}

// Send is a helper method to define mock.On call
//   - c tgbotapi.Chattable
func (_e *MocktgClient_Expecter) Send(c interface{}) *MocktgClient_Send_Call {
	return &MocktgClient_Send_Call{Call: _e.mock.On("Send", c)}
}

func (_c *MocktgClient_Send_Call) Run(run func(c tgbotapi.Chattable)) *MocktgClient_Send_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(tgbotapi.Chattable))
	})
	return _c
}

func (_c *MocktgClient_Send_Call) Return(_a0 tgbotapi.Message, _a1 error) *MocktgClient_Send_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MocktgClient_Send_Call) RunAndReturn(run func(tgbotapi.Chattable) (tgbotapi.Message, error)) *MocktgClient_Send_Call {
	_c.Call.Return(run)
	return _c
}

// StopReceivingUpdates provides a mock function with no fields
func (_m *MocktgClient) StopReceivingUpdates() {
	_m.Called()
}

// MocktgClient_StopReceivingUpdates_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'StopReceivingUpdates'
type MocktgClient_StopReceivingUpdates_Call struct {
	*mock.Call
}

// StopReceivingUpdates is a helper method to define mock.On call
func (_e *MocktgClient_Expecter) StopReceivingUpdates() *MocktgClient_StopReceivingUpdates_Call {
	return &MocktgClient_StopReceivingUpdates_Call{Call: _e.mock.On("StopReceivingUpdates")}
}

func (_c *MocktgClient_StopReceivingUpdates_Call) Run(run func()) *MocktgClient_StopReceivingUpdates_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MocktgClient_StopReceivingUpdates_Call) Return() *MocktgClient_StopReceivingUpdates_Call {
	_c.Call.Return()
	return _c
}

func (_c *MocktgClient_StopReceivingUpdates_Call) RunAndReturn(run func()) *MocktgClient_StopReceivingUpdates_Call {
	_c.Run(run)
	return _c
}

// NewMocktgClient creates a new instance of MocktgClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMocktgClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *MocktgClient {
	mock := &MocktgClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
