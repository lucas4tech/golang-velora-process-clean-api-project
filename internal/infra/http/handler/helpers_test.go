package handler

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	apperrors "rankmyapp/pkg/errors"
)

func TestWrapBindError(t *testing.T) {
	err := errors.New("json: invalid")
	wrapped := wrapBindError(err)
	assert.Error(t, wrapped)
	appErr, ok := apperrors.AsAppError(wrapped)
	assert.True(t, ok)
	assert.Equal(t, apperrors.ErrInvalidItem.Code, appErr.Code)
	assert.Equal(t, 400, appErr.StatusCode)
}

func TestErrInvalidID(t *testing.T) {
	err := errInvalidID()
	assert.Error(t, err)
	appErr, ok := apperrors.AsAppError(err)
	assert.True(t, ok)
	assert.Equal(t, apperrors.ErrInvalidOrderID.Code, appErr.Code)
}
