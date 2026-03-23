package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	_ "rankmyapp/docs"
	"rankmyapp/internal/infra/http/handler"
)

func TestSetup_Health(t *testing.T) {
	h := handler.NewOrderHandler(nil, nil, nil, nil)
	r := Setup(h)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestSetup_SwaggerMount(t *testing.T) {
	h := handler.NewOrderHandler(nil, nil, nil, nil)
	r := Setup(h)

	req := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSetup_SwaggerDocJSON(t *testing.T) {
	h := handler.NewOrderHandler(nil, nil, nil, nil)
	r := Setup(h)

	req := httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, w.Body.String(), `"swagger"`)
}
