package handler

import (
	"net/http"

	apperrors "rankmyapp/pkg/errors"
)

func wrapBindError(err error) error {
	return apperrors.Wrap(
		apperrors.ErrInvalidItem.Code,
		"invalid input data: "+err.Error(),
		http.StatusBadRequest,
		err,
	)
}

func errInvalidID() error {
	return apperrors.ErrInvalidOrderID
}
