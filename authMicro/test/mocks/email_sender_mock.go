package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockEmailSender is a mock type for the EmailSender interface
type MockEmailSender struct {
	mock.Mock
}

type MockEmailSender_Expecter struct {
	mock *mock.Mock
}

func (_m *MockEmailSender) EXPECT() *MockEmailSender_Expecter {
	return &MockEmailSender_Expecter{mock: &_m.Mock}
}

// SendVerificationCode provides a mock function with given fields: ctx, to, code, lang
func (_m *MockEmailSender) SendVerificationCode(ctx context.Context, to, code, lang string) error {
	ret := _m.Called(ctx, to, code, lang)

	if len(ret) == 0 {
		panic("no return value specified for SendVerificationCode")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string) error); ok {
		r0 = rf(ctx, to, code, lang)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockEmailSender_SendVerificationCode_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SendVerificationCode'
type MockEmailSender_SendVerificationCode_Call struct {
	*mock.Call
}

func (_e *MockEmailSender_Expecter) SendVerificationCode(ctx, to, code, lang interface{}) *MockEmailSender_SendVerificationCode_Call {
	return &MockEmailSender_SendVerificationCode_Call{Call: _e.mock.On("SendVerificationCode", ctx, to, code, lang)}
}

func (_c *MockEmailSender_SendVerificationCode_Call) Run(run func(ctx context.Context, to, code, lang string)) *MockEmailSender_SendVerificationCode_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string), args[3].(string))
	})
	return _c
}

func (_c *MockEmailSender_SendVerificationCode_Call) Return(_a0 error) *MockEmailSender_SendVerificationCode_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockEmailSender_SendVerificationCode_Call) RunAndReturn(run func(context.Context, string, string, string) error) *MockEmailSender_SendVerificationCode_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockEmailSender creates a new instance of MockEmailSender.
func NewMockEmailSender(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockEmailSender {
	mock := &MockEmailSender{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
