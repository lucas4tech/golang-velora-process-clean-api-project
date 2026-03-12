package errors

import (
	"errors"
	"fmt"
	"net/http"
)

type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
	Err        error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

func New(code, message string, statusCode int) *AppError {
	return &AppError{Code: code, Message: message, StatusCode: statusCode}
}

func Wrap(code, message string, statusCode int, err error) *AppError {
	return &AppError{Code: code, Message: message, StatusCode: statusCode, Err: err}
}

var (
	ErrOrderNotFound     = New("ORDER_NOT_FOUND", "order not found", http.StatusNotFound)
	ErrInvalidOrderID    = New("INVALID_ORDER_ID", "invalid order ID", http.StatusBadRequest)
	ErrInvalidStatus     = New("INVALID_STATUS", "invalid status", http.StatusBadRequest)
	ErrInvalidTransition = New("INVALID_TRANSITION", "invalid status transition", http.StatusUnprocessableEntity)
	ErrEmptyItems        = New("EMPTY_ITEMS", "order must have at least one item", http.StatusBadRequest)
	ErrInvalidItem       = New("INVALID_ITEM", "invalid item", http.StatusBadRequest)
	ErrInternal          = New("INTERNAL_ERROR", "internal server error", http.StatusInternalServerError)
)

func AsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	return appErr, errors.As(err, &appErr)
}
