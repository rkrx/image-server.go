package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"image-server/src/tools"
)

func newCORSBackend(t *testing.T, cors tools.CORSConfig) *Backend {
	t.Helper()
	cacheDir := t.TempDir()
	return NewBackend(tools.Config{
		DisableHashChecking: true,
		ImageOriginalDir:    filepath.Join(cacheDir, "originals"),
		ImageCacheDir:       filepath.Join(cacheDir, "images"),
		Server:              tools.ServerType{OriginalBackend: "http://127.0.0.1:0"},
		CORS:                cors,
	})
}

func TestCORSPreflightReturnsConfiguredHeaders(t *testing.T) {
	backend := newCORSBackend(t, tools.CORSConfig{
		Enabled:         true,
		AllowedOrigins:  []string{"https://app.example.com"},
		AllowedMethods:  []string{"GET", "HEAD"},
		AllowedHeaders:  []string{"Accept"},
		ExposedHeaders:  []string{"Content-Length"},
		MaxAgeInSeconds: 600,
	})

	request := httptest.NewRequest(http.MethodOptions, "/image.jpg", nil)
	request.Header.Set("Origin", "https://app.example.com")
	request.Header.Set("Access-Control-Request-Method", "GET")
	response := httptest.NewRecorder()

	backend.ServeHTTP(response, request)

	result := response.Result()
	defer result.Body.Close()
	assert.Equal(t, http.StatusNoContent, result.StatusCode)
	assert.Equal(t, "https://app.example.com", result.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, HEAD", result.Header.Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Accept", result.Header.Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "Content-Length", result.Header.Get("Access-Control-Expose-Headers"))
	assert.Equal(t, "600", result.Header.Get("Access-Control-Max-Age"))
	assert.Contains(t, result.Header.Values("Vary"), "Origin")
}

func TestCORSPreflightRejectsUnknownOrigin(t *testing.T) {
	backend := newCORSBackend(t, tools.CORSConfig{
		Enabled:        true,
		AllowedOrigins: []string{"https://app.example.com"},
	})

	request := httptest.NewRequest(http.MethodOptions, "/image.jpg", nil)
	request.Header.Set("Origin", "https://evil.example.com")
	request.Header.Set("Access-Control-Request-Method", "GET")
	response := httptest.NewRecorder()

	backend.ServeHTTP(response, request)

	result := response.Result()
	defer result.Body.Close()
	assert.Equal(t, http.StatusForbidden, result.StatusCode)
	assert.Empty(t, result.Header.Get("Access-Control-Allow-Origin"))
}

func TestCORSWildcardOriginWithoutCredentials(t *testing.T) {
	backend := newCORSBackend(t, tools.CORSConfig{
		Enabled:        true,
		AllowedOrigins: []string{"*"},
	})

	request := httptest.NewRequest(http.MethodOptions, "/image.jpg", nil)
	request.Header.Set("Origin", "https://whatever.example.com")
	request.Header.Set("Access-Control-Request-Method", "GET")
	response := httptest.NewRecorder()

	backend.ServeHTTP(response, request)

	result := response.Result()
	defer result.Body.Close()
	assert.Equal(t, http.StatusNoContent, result.StatusCode)
	assert.Equal(t, "*", result.Header.Get("Access-Control-Allow-Origin"))
	assert.NotContains(t, result.Header.Values("Vary"), "Origin")
}

func TestCORSSimpleRequestHeaders(t *testing.T) {
	backend := newCORSBackend(t, tools.CORSConfig{
		Enabled:          true,
		AllowedOrigins:   []string{"https://app.example.com"},
		AllowCredentials: true,
	})

	request := httptest.NewRequest(http.MethodGet, "/image.jpg", nil)
	request.Header.Set("Origin", "https://app.example.com")
	response := httptest.NewRecorder()

	backend.ServeHTTP(response, request)

	assert.Equal(t, "https://app.example.com", response.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", response.Header().Get("Access-Control-Allow-Credentials"))
	assert.Contains(t, response.Header().Values("Vary"), "Origin")
}

func TestCORSDisabledAddsNoHeaders(t *testing.T) {
	backend := newCORSBackend(t, tools.CORSConfig{})

	request := httptest.NewRequest(http.MethodOptions, "/image.jpg", nil)
	request.Header.Set("Origin", "https://app.example.com")
	request.Header.Set("Access-Control-Request-Method", "GET")
	response := httptest.NewRecorder()

	backend.ServeHTTP(response, request)

	result := response.Result()
	defer result.Body.Close()
	assert.Equal(t, http.StatusMethodNotAllowed, result.StatusCode)
	assert.Empty(t, result.Header.Get("Access-Control-Allow-Origin"))
}
