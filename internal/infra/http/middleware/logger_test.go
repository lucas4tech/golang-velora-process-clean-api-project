package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRequestLogger_NoErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

	RequestLogger()(ctx)
	ctx.Next()

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequestLogger_WithErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	_ = ctx.Error(assert.AnError)

	RequestLogger()(ctx)
	ctx.Next()

	assert.Equal(t, http.StatusOK, w.Code)
}
