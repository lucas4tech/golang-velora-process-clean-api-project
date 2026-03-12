package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	apperrors "rankmyapp/pkg/errors"
)

func TestErrorHandler_NoErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandler())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestErrorHandler_AppError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandler())
	r.GET("/", func(c *gin.Context) { _ = c.Error(apperrors.ErrOrderNotFound) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "ORDER_NOT_FOUND")
}

func TestErrorHandler_NonAppError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandler())
	r.GET("/", func(c *gin.Context) { _ = c.Error(assert.AnError) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "INTERNAL_ERROR")
}
