package server

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image-server/src/tools"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type imageHTTPError struct {
	status int
	err    error
}

func (e *imageHTTPError) Error() string { return e.err.Error() }
func (e *imageHTTPError) Unwrap() error { return e.err }

func imageErrorStatus(err error, fallback int) int {
	var statusError *imageHTTPError
	if errors.As(err, &statusError) {
		return statusError.status
	}
	return fallback
}

func (backend *Backend) loadSourceImage(ctx context.Context, requestPath, filename string, profile tools.Profile, budget *compositionBudget) (image.Image, int, error) {
	if err := os.MkdirAll(profile.ImageCacheDir, 0750); err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if err := os.Chmod(profile.ImageCacheDir, 0750); err != nil {
		return nil, http.StatusInternalServerError, err
	}
	cachedPath := filepath.Join(profile.ImageCacheDir, filename)
	if !tools.FileExists(cachedPath) {
		content, status, err := backend.FetchImage(ctx, requestPath)
		if err != nil {
			return nil, status, err
		}
		// Different layouts can fetch the same child concurrently. Publishing
		// complete files avoids readers seeing a partially written original.
		if err := writeSourceCacheFile(cachedPath, content); err != nil {
			return nil, http.StatusInternalServerError, err
		}
	}
	file, err := os.Open(cachedPath)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	defer file.Close()
	if err := file.Chmod(0600); err != nil {
		return nil, http.StatusInternalServerError, err
	}
	maxBytes := backend.originMaxImageBytes()
	content, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if int64(len(content)) > maxBytes {
		return nil, http.StatusRequestEntityTooLarge, errors.New("cached origin image too large")
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil {
		return nil, http.StatusUnsupportedMediaType, err
	}
	if err := backend.Config.ValidateSourceImageDimensions(config.Width, config.Height); err != nil {
		return nil, http.StatusRequestEntityTooLarge, err
	}
	if err := budget.chargeSource(config.Width, config.Height); err != nil {
		return nil, http.StatusRequestEntityTooLarge, err
	}
	img, _, err := image.Decode(bytes.NewReader(content))
	if err != nil {
		return nil, http.StatusUnsupportedMediaType, err
	}
	return img, http.StatusOK, nil
}

func writeSourceCacheFile(path string, content []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".source-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(content); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
