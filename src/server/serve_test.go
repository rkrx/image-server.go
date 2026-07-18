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
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

func TestBackendServeHTTPRejectsUnsupportedMethods(t *testing.T) {
	backend := NewBackend(tools.Config{})
	request := httptest.NewRequest(http.MethodPost, "/source.png", nil)
	response := httptest.NewRecorder()

	backend.ServeHTTP(response, request)

	assert.Equal(t, http.StatusMethodNotAllowed, response.Code)
	assert.Equal(t, "GET, HEAD", response.Header().Get("Allow"))
	assert.Equal(t, "nosniff", response.Header().Get("X-Content-Type-Options"))
}

func TestBackendServeHTTPDoesNotReflectInvalidOutputType(t *testing.T) {
	backend := NewBackend(tools.Config{DisableHashChecking: true})
	request := httptest.NewRequest(http.MethodGet, "/source.png?type=%3Cscript%3Ealert(1)%3C/script%3E", nil)
	response := httptest.NewRecorder()

	backend.ServeHTTP(response, request)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.NotContains(t, response.Body.String(), "<script>")
	assert.Equal(t, "text/plain; charset=utf-8", response.Header().Get("Content-Type"))
	assert.Equal(t, "nosniff", response.Header().Get("X-Content-Type-Options"))
}

func TestBackendServeHTTPRejectsSourceDimensionsOverLimit(t *testing.T) {
	sourceImage := image.NewRGBA(image.Rect(0, 0, 16, 12))
	var sourceImageBytes bytes.Buffer
	require.NoError(t, png.Encode(&sourceImageBytes, sourceImage))

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(sourceImageBytes.Bytes())
	}))
	defer origin.Close()

	cacheDir := t.TempDir()
	backend := NewBackend(tools.Config{
		DisableHashChecking:  true,
		ImageCacheDir:        filepath.Join(cacheDir, "images"),
		MaxSourceImageWidth:  8,
		MaxSourceImageHeight: 100,
		MaxSourceImagePixels: 1000,
		MaxOutputImageWidth:  100,
		MaxOutputImageHeight: 100,
		MaxOutputImagePixels: 1000,
		Server:               tools.ServerType{OriginalBackend: origin.URL},
		JpegDefaults:         tools.JpegDefaults{Quality: tools.DefaultJpegQuality},
	})
	request := httptest.NewRequest(http.MethodGet, "/source.png", nil)
	response := httptest.NewRecorder()

	backend.ServeHTTP(response, request)

	assert.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
}

func TestBackendServeHTTPRejectsDerivedOutputDimensionsOverLimit(t *testing.T) {
	sourceImage := image.NewRGBA(image.Rect(0, 0, 100, 1))
	var sourceImageBytes bytes.Buffer
	require.NoError(t, png.Encode(&sourceImageBytes, sourceImage))

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(sourceImageBytes.Bytes())
	}))
	defer origin.Close()

	cacheDir := t.TempDir()
	backend := NewBackend(tools.Config{
		DisableHashChecking:  true,
		ImageCacheDir:        filepath.Join(cacheDir, "images"),
		MaxSourceImageWidth:  100,
		MaxSourceImageHeight: 100,
		MaxSourceImagePixels: 10000,
		MaxOutputImageWidth:  1000,
		MaxOutputImageHeight: 1000,
		MaxOutputImagePixels: 1000000,
		Server:               tools.ServerType{OriginalBackend: origin.URL},
		JpegDefaults:         tools.JpegDefaults{Quality: tools.DefaultJpegQuality},
	})
	request := httptest.NewRequest(http.MethodGet, "/source.png?height=100", nil)
	response := httptest.NewRecorder()

	backend.ServeHTTP(response, request)

	assert.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
}

func TestBackendServeHTTPEnforcesRequestLimits(t *testing.T) {
	backend := NewBackend(tools.Config{
		DisableHashChecking: true,
		Server: tools.ServerType{
			MaxRequestURIBytes: 20,
			MaxQueryParameters: 2,
		},
	})

	longRequest := httptest.NewRequest(http.MethodGet, "/source.png?value=1234567890", nil)
	longResponse := httptest.NewRecorder()
	backend.ServeHTTP(longResponse, longRequest)
	assert.Equal(t, http.StatusRequestURITooLong, longResponse.Code)

	queryRequest := httptest.NewRequest(http.MethodGet, "/x?a=1&b=2&c=3", nil)
	queryResponse := httptest.NewRecorder()
	backend.ServeHTTP(queryResponse, queryRequest)
	assert.Equal(t, http.StatusBadRequest, queryResponse.Code)
}

func TestBackendServeHTTPSharesConcurrentCacheMisses(t *testing.T) {
	sourceImage := image.NewRGBA(image.Rect(0, 0, 16, 12))
	var sourceImageBytes bytes.Buffer
	require.NoError(t, png.Encode(&sourceImageBytes, sourceImage))

	var originRequests atomic.Int32
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originRequests.Add(1)
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write(sourceImageBytes.Bytes())
	}))
	defer origin.Close()

	cacheDir := t.TempDir()
	backend := NewBackend(tools.Config{
		DisableHashChecking: true,
		ImageCacheDir:       filepath.Join(cacheDir, "images"),
		Server: tools.ServerType{
			OriginalBackend:        origin.URL,
			ImageProcessingWorkers: 1,
		},
		JpegDefaults: tools.JpegDefaults{Quality: tools.DefaultJpegQuality},
	})

	const requestCount = 8
	var waitGroup sync.WaitGroup
	statusCodes := make(chan int, requestCount)
	for range requestCount {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			request := httptest.NewRequest(http.MethodGet, "/source.png?type=jpeg", nil)
			response := httptest.NewRecorder()
			backend.ServeHTTP(response, request)
			statusCodes <- response.Code
		}()
	}
	waitGroup.Wait()
	close(statusCodes)

	for statusCode := range statusCodes {
		assert.Equal(t, http.StatusOK, statusCode)
	}
	assert.Equal(t, int32(1), originRequests.Load())
	require.NoError(t, filepath.WalkDir(backend.Config.ImageCacheDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		fileInfo, statErr := entry.Info()
		if statErr != nil {
			return statErr
		}
		assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm(), path)
		return nil
	}))
}
