package server

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"image-server/src/tools"
)

func TestBackendServeHTTPNegotiatesAndEncodesOutputFormats(t *testing.T) {
	sourceImage := image.NewRGBA(image.Rect(0, 0, 16, 12))
	for y := 0; y < sourceImage.Bounds().Dy(); y++ {
		for x := 0; x < sourceImage.Bounds().Dx(); x++ {
			sourceImage.Set(x, y, color.RGBA{R: uint8(x * 12), G: uint8(y * 16), B: 100, A: 255})
		}
	}
	var sourceImageBytes bytes.Buffer
	require.NoError(t, png.Encode(&sourceImageBytes, sourceImage))

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(sourceImageBytes.Bytes())
	}))
	defer origin.Close()

	cacheDir := t.TempDir()
	backend := NewBackend(tools.Config{
		DisableHashChecking: true,
		ImageOriginalDir:    filepath.Join(cacheDir, "originals"),
		ImageCacheDir:       filepath.Join(cacheDir, "images"),
		Server:              tools.ServerType{OriginalBackend: origin.URL},
		JpegDefaults:        tools.JpegDefaults{Quality: tools.DefaultJpegQuality},
		WebpDefaults:        tools.WebpDefaults{Quality: tools.DefaultWebpQuality},
		AvifDefaults: tools.AvifDefaults{
			Quality:      tools.DefaultAvifQuality,
			AlphaQuality: tools.DefaultAvifAlphaQuality,
			Speed:        tools.DefaultAvifSpeed,
		},
	})

	testCases := []struct {
		name           string
		requestTarget  string
		accept         string
		expectedFormat string
		expectedType   string
		negotiated     bool
	}{
		{name: "negotiated avif", requestTarget: "/source.png", accept: "image/avif,image/webp", expectedFormat: "avif", expectedType: "image/avif", negotiated: true},
		{name: "negotiated webp", requestTarget: "/source.png", accept: "image/webp", expectedFormat: "webp", expectedType: "image/webp", negotiated: true},
		{name: "explicit jpeg", requestTarget: "/source.png?type=jpeg", accept: "image/avif,image/webp", expectedFormat: "jpeg", expectedType: "image/jpeg", negotiated: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, testCase.requestTarget, nil)
			request.Header.Set("Accept", testCase.accept)
			response := httptest.NewRecorder()

			backend.ServeHTTP(response, request)

			result := response.Result()
			defer result.Body.Close()
			assert.Equal(t, http.StatusOK, result.StatusCode)
			assert.Equal(t, testCase.expectedType, result.Header.Get("Content-Type"))
			if testCase.negotiated {
				assert.Contains(t, result.Header.Values("Vary"), "Accept")
			} else {
				assert.NotContains(t, result.Header.Values("Vary"), "Accept")
			}

			_, format, err := image.Decode(result.Body)
			assert.NoError(t, err)
			assert.Equal(t, testCase.expectedFormat, format)
		})
	}

	cacheEntries := make([]string, 0)
	require.NoError(t, filepath.WalkDir(backend.Config.ImageCacheDir, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			cacheEntries = append(cacheEntries, path)
		}
		return err
	}))
	assert.True(t, containsFileWithSuffix(cacheEntries, ".avif"))
	assert.True(t, containsFileWithSuffix(cacheEntries, ".webp"))
	assert.True(t, containsFileWithSuffix(cacheEntries, ".jpeg"))
}

func containsFileWithSuffix(filenames []string, suffix string) bool {
	for _, filename := range filenames {
		if strings.HasSuffix(filename, suffix) {
			return true
		}
	}
	return false
}
