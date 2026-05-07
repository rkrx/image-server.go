package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

func (backend *Backend) FetchImage(ctx context.Context, fetchUrl string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchUrl, nil)
	if err != nil {
		return nil, 500, err
	}

	resp, err := backend.originHTTPClient().Do(req)
	if err != nil {
		return nil, 502, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, errors.New("invalid status code")
	}

	maxBytes := backend.originMaxImageBytes()
	if resp.ContentLength > maxBytes {
		return nil, 413, fmt.Errorf("origin image too large: %d bytes exceeds %d bytes", resp.ContentLength, maxBytes)
	}

	limitedReader := &io.LimitedReader{R: resp.Body, N: maxBytes + 1}
	imgBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, 502, err
	}
	if int64(len(imgBytes)) > maxBytes {
		return nil, 413, fmt.Errorf("origin image too large: exceeds %d bytes", maxBytes)
	}

	return imgBytes, resp.StatusCode, nil
}

func (backend *Backend) originHTTPClient() *http.Client {
	if backend.HTTPClient != nil {
		return backend.HTTPClient
	}
	backend.HTTPClient = newOriginHTTPClient(backend.Config)
	return backend.HTTPClient
}

func (backend *Backend) originMaxImageBytes() int64 {
	if backend.Config.OriginMaxImageBytes > 0 {
		return backend.Config.OriginMaxImageBytes
	}
	return 50 << 20
}
