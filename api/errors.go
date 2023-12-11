package api

import (
	"errors"
)

var PushEventError = errors.New("Push event error")

type pushEventError struct {
	msg string
}

func NewPushEventError(message string) *pushEventError {
	return &pushEventError{msg: message}
}

func (e *pushEventError) Error() string  { return e.msg }
func (e *pushEventError) ErrorCode() int { return -62301 }

type requestCanceledError struct {
	msg string
}

func NewRequestCanceledError(message string) *requestCanceledError {
	return &requestCanceledError{msg: message}
}

func (e *requestCanceledError) Error() string  { return e.msg }
func (e *requestCanceledError) ErrorCode() int { return -62302 }
