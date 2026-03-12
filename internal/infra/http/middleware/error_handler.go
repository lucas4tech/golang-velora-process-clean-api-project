package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"rankmyapp/internal/app/dto"
	apperrors "rankmyapp/pkg/errors"
	"rankmyapp/pkg/logger"
)

func ErrorHandler() gin.HandlerFunc {
	log := logger.Get()
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err
		if appErr, ok := apperrors.AsAppError(err); ok {
			c.JSON(appErr.StatusCode, dto.ErrorResponse{
				Code:    appErr.Code,
				Message: appErr.Message,
			})
			return
		}

		log.Error("unhandled error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "internal server error",
		})
	}
}
