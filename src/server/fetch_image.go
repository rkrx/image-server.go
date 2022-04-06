package server

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (backend *Backend) FetchImage(path string) ([]byte, int, error) {
	backendStr := strings.TrimRight(backend.Config.Server.OriginalBackend, "/")
	path = strings.TrimLeft(path, "/")
	fetchUrl := fmt.Sprintf("%s/%s", backendStr, path)
	resp, err := http.Get(fetchUrl)
	if err != nil {
		return nil, 500, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		imgBytes, err := io.ReadAll(resp.Body)
		return imgBytes, resp.StatusCode, err
	}
	return nil, resp.StatusCode, errors.New("invalid status code")
}
