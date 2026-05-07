package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"image-server/src/tools"
)

func TestBackend_FetchImage(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("image-bytes"))
	}))
	defer origin.Close()

	backend := NewBackend(tools.Config{
		OriginFetchTimeoutInSeconds: 15,
		OriginMaxImageBytes:         50 << 20,
	})

	imageBytes, statusCode, err := backend.FetchImage(context.Background(), origin.URL)

	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, statusCode)
	assert.Equal(t, []byte("image-bytes"), imageBytes)
}

func TestBackend_FetchImageRejectsContentLengthOverLimit(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "11")
		_, _ = w.Write([]byte("image-bytes"))
	}))
	defer origin.Close()

	backend := NewBackend(tools.Config{
		OriginFetchTimeoutInSeconds: 15,
		OriginMaxImageBytes:         10,
	})

	imageBytes, statusCode, err := backend.FetchImage(context.Background(), origin.URL)

	assert.Nil(t, imageBytes)
	assert.Equal(t, http.StatusRequestEntityTooLarge, statusCode)
	assert.Error(t, err)
}

func TestBackend_FetchImageRejectsChunkedResponseOverLimit(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.(http.Flusher).Flush()
		_, _ = w.Write([]byte("image-bytes"))
	}))
	defer origin.Close()

	backend := NewBackend(tools.Config{
		OriginFetchTimeoutInSeconds: 15,
		OriginMaxImageBytes:         10,
	})

	imageBytes, statusCode, err := backend.FetchImage(context.Background(), origin.URL)

	assert.Nil(t, imageBytes)
	assert.Equal(t, http.StatusRequestEntityTooLarge, statusCode)
	assert.Error(t, err)
}

func TestBackend_FetchImageUsesContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	backend := NewBackend(tools.Config{
		OriginFetchTimeoutInSeconds: 15,
		OriginMaxImageBytes:         50 << 20,
	})

	imageBytes, statusCode, err := backend.FetchImage(ctx, "http://127.0.0.1/")

	assert.Nil(t, imageBytes)
	assert.Equal(t, http.StatusBadGateway, statusCode)
	assert.Error(t, err)
}
