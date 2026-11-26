package server

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	"image-server/src/imaging"
	"image-server/src/tools"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"
)

type Backend struct {
	Config          tools.Config
	HTTPClient      *http.Client
	processingSlots chan struct{}
	processingGroup *singleflight.Group
	corsMatcher     corsOriginMatcher
}

func NewBackend(config tools.Config) *Backend {
	workers := config.Server.ImageProcessingWorkers
	if workers < 1 {
		workers = tools.DefaultImageProcessingWorkers
	}
	return &Backend{
		Config:          config,
		HTTPClient:      newOriginHTTPClient(config),
		processingSlots: make(chan struct{}, workers),
		processingGroup: &singleflight.Group{},
		corsMatcher:     compileCORSOrigins(config.CORS.AllowedOrigins),
	}
}

func newOriginHTTPClient(config tools.Config) *http.Client {
	timeout := config.OriginFetchTimeoutInSeconds
	if timeout < 1 {
		timeout = 15
	}

	return &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}

func (backend *Backend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if applyCORS(w, r, backend.Config.CORS, backend.corsMatcher) {
		return
	}
	w = imageCacheWriter{ResponseWriter: w, config: &backend.Config}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		allow := "GET, HEAD"
		if backend.Config.CORS.Enabled {
			allow = "GET, HEAD, OPTIONS"
		}
		w.Header().Set("Allow", allow)
		writeHTTPError(w, http.StatusMethodNotAllowed)
		return
	}
	if len(r.RequestURI) > positiveOrDefault(backend.Config.Server.MaxRequestURIBytes, tools.DefaultMaxRequestURIBytes) {
		writeHTTPError(w, http.StatusRequestURITooLong)
		return
	}
	if r.URL.RawQuery != "" && strings.Count(r.URL.RawQuery, "&")+1 > positiveOrDefault(backend.Config.Server.MaxQueryParameters, tools.DefaultMaxQueryParameters) {
		writeHTTPError(w, http.StatusBadRequest)
		return
	}

	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		writeHTTPError(w, http.StatusBadRequest)
		return
	}
	profile := backend.Config.GetDefaultProfile()

	//region Check if the client ip found in DisableHashCheckingFromIPv4Sources
	getRemoteIP := func(header http.Header, remoteAddrHeader string) string {
		if remoteAddrHeader != "" {
			if ip := r.Header.Get(remoteAddrHeader); ip != "" {
				if success, _ := tools.RegexMatch(ip, "^(?:\\[[(:0-9a-fA-F)]+\\]|(\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3})):\\d+$"); success {
					ipAddr, _, err := net.SplitHostPort(ip)
					if err != nil {
						return ""
					}
					return net.ParseIP(ipAddr).String()
				}
				return net.ParseIP(ip).String()
			}
		} else {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err == nil {
				return net.ParseIP(ip).String()
			}
		}
		return ""
	}
	remoteAddrIp := getRemoteIP(r.Header, backend.Config.Server.RemoteAddrHeader)
	disableHashChecking := backend.Config.DisableHashChecking
	if remoteAddrIp != "" && tools.Contains(backend.Config.DisableHashCheckingFromIPv4Sources, remoteAddrIp) {
		disableHashChecking = true
	}
	//endregion

	//region Apply URL-Parameters to the current profile
	if err := applyUrlParameters(&profile, backend.Config, r.RequestURI, values, disableHashChecking); err != nil {
		if errors.Is(err, errInvalidRequestHash) {
			writeHTTPError(w, http.StatusForbidden)
		} else {
			writeHTTPError(w, http.StatusBadRequest)
		}
		return
	}
	//endregion

	if profile.Script != "" {
		if values.Has("put") || values.Has("putb") || values.Has("canvas") {
			writeHTTPError(w, http.StatusBadRequest)
			return
		}
		if profile.LayoutScript == nil {
			writeHTTPError(w, http.StatusInternalServerError)
			return
		}
		if err := backend.acquireProcessingSlot(r.Context()); err != nil {
			writeHTTPError(w, http.StatusRequestTimeout)
			return
		}
		pathname := "/" + strings.TrimPrefix(r.URL.Path, "/")
		result, err := profile.LayoutScript.Run(r.Context(), pathname, values)
		backend.releaseProcessingSlot()
		if err != nil || backend.Config.ValidateOutputImageDimensions(result.Width, result.Height) != nil {
			writeHTTPError(w, http.StatusBadRequest)
			return
		}
		profile.Width, profile.Height = result.Width, result.Height
		if result.Canvas != "" {
			values.Set("canvas", result.Canvas)
		}
		values.Set("put", string(result.Layout))
	}

	virtualCanvas := values.Has("canvas")
	if virtualCanvas && (len(values["canvas"]) != 1 || values.Get("canvas") != "white" || profile.Width < 1 || profile.Height < 1) {
		writeHTTPError(w, http.StatusBadRequest)
		return
	}

	var composition *compositionPlan
	if values.Has("put") || values.Has("putb") {
		payload, err := compositionPayload(values)
		if err != nil {
			writeHTTPError(w, http.StatusBadRequest)
			return
		}
		composition, err = backend.prepareComposition(payload, profile)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, imaging.ErrCompositionLimit) {
				status = http.StatusRequestEntityTooLarge
			}
			writeHTTPError(w, status)
			return
		}
	}

	//region Determine the output file type
	negotiated, err := applyOutputFileType(&profile, r)
	if err != nil {
		writeHTTPError(w, http.StatusBadRequest)
		return
	}
	if negotiated {
		w.Header().Add("Vary", "Accept")
	}
	w.Header().Set("Content-Type", profile.Type.ContentType())
	//endregion

	//region Project the origFilename and the cachePath
	origFilename, cachePath := "", ""
	sourcePath := r.URL.EscapedPath()
	cacheKey := profile.GetDirectoryHash(backend.Config)
	if virtualCanvas {
		cacheKey += ";canvas=white"
	}
	originalKey := cacheKey
	if composition != nil {
		// Keep directory components short, independently of profile-name length.
		cacheKey = fmt.Sprintf("composition-v1-%x", sha256.Sum256([]byte(cacheKey+";put="+composition.cacheHash)))
	}
	if backend.Config.HashCacheFilenames {
		origFilename = tools.Md5(filepath.Join(originalKey, sourcePath))
		cacheFilename := tools.Md5(filepath.Join(cacheKey, sourcePath))
		cachePath = filepath.Join(profile.ImageCacheDir, fmt.Sprintf("%s.%s", cacheFilename, profile.Type))
	} else {
		origFilename = tools.Md5(sourcePath)
		cachePath = filepath.Join(profile.ImageCacheDir, cacheKey, fmt.Sprintf("%s.%s", origFilename, profile.Type))
	}
	requestPath := tools.ConcatPathParts(profile.OriginalBackend, sourcePath)
	if composition != nil {
		origFilename = fmt.Sprintf("source-%x", sha256.Sum256([]byte(requestPath)))
	}

	if composition == nil {
		if success, groups := tools.RegexMatch(r.URL.Path, "\\.(jpe?g|gif|png|webp|avif)$"); success {
			origFilename = fmt.Sprintf("%s.%s", origFilename, groups[1])
		}
	}
	//endregion

	//region Generate the cache image
	if !tools.FileExists(cachePath) {
		log.Print("File not in cache")

		resultChannel := backend.processingGroup.DoChan(cachePath, func() (any, error) {
			if tools.FileExists(cachePath) {
				return imageProcessingResult{statusCode: http.StatusOK}, nil
			}
			if err := backend.acquireProcessingSlot(r.Context()); err != nil {
				return imageProcessingResult{statusCode: http.StatusRequestTimeout}, err
			}
			defer backend.releaseProcessingSlot()

			if tools.FileExists(cachePath) {
				return imageProcessingResult{statusCode: http.StatusOK}, nil
			}

			var budget *compositionBudget
			if composition != nil || virtualCanvas {
				budget = newCompositionBudget(backend.Config)
			}
			statusCode, processingErr := backend.getFinalImageBytes(r.Context(), requestPath, origFilename, cachePath, profile, budget, virtualCanvas, func(img image.Image) (image.Image, error) {
				var err error
				if !virtualCanvas {
					img, err = backend.transformImage(img, profile, composition == nil, budget)
					if err != nil {
						return nil, err
					}
				}
				if composition != nil {
					img, err = backend.composeImage(r.Context(), img, composition, budget)
					if err != nil {
						return nil, err
					}
				}
				if (composition != nil || virtualCanvas) && profile.Sharpen > 0 {
					if err := budget.chargeRender(img.Bounds().Dx(), img.Bounds().Dy()); err != nil {
						return nil, err
					}
					img = imaging.Sharpen(img, float64(profile.Sharpen))
				}
				return img, nil
			})
			return imageProcessingResult{statusCode: statusCode}, processingErr
		})

		var processingResult singleflight.Result
		select {
		case <-r.Context().Done():
			return
		case processingResult = <-resultChannel:
		}
		result, ok := processingResult.Val.(imageProcessingResult)
		if !ok {
			result.statusCode = http.StatusInternalServerError
		}
		if processingResult.Err != nil {
			log.Printf("Image processing failed with status %d", result.statusCode)
			writeHTTPError(w, result.statusCode)
			return
		}
	} else {
		log.Print("File in cache")
	}
	//endregion

	//region Write the cached image contents to the client result
	// #nosec G302 -- cache directories require execute permission for traversal.
	if err := os.Chmod(filepath.Dir(cachePath), 0750); err != nil {
		log.Print("Could not secure cache directory")
		writeHTTPError(w, http.StatusInternalServerError)
		return
	}
	file, err := os.Open(cachePath)
	if err != nil {
		log.Print("Could not open cache file")
		writeHTTPError(w, http.StatusInternalServerError)
		return
	}
	defer file.Close()
	if err := file.Chmod(0600); err != nil {
		log.Print("Could not secure cache file")
		writeHTTPError(w, http.StatusInternalServerError)
		return
	}
	fileInfo, err := file.Stat()
	if err != nil {
		log.Print("Could not stat cache file")
		writeHTTPError(w, http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, "", fileInfo.ModTime(), file)
	//endregion
}

type imageCacheWriter struct {
	http.ResponseWriter
	config *tools.Config
}

// ServeContent can select a different status after checking request preconditions.
func (w imageCacheWriter) WriteHeader(status int) {
	maxAge := w.config.CacheMaxAgeInSeconds
	switch {
	case status == http.StatusNotFound && !w.config.DisableCaching && maxAge > 0:
		w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", min(maxAge, 60)))
	case status >= 400:
		w.Header().Set("Cache-Control", "no-store")
	case status == http.StatusOK || status == http.StatusPartialContent || status == http.StatusNotModified:
		if w.config.DisableCaching {
			w.Header().Set("Cache-Control", "no-cache")
		} else if maxAge > 0 {
			w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", maxAge))
		}
	}
	w.ResponseWriter.WriteHeader(status)
}

// Preserve the underlying HTTP writer's optimized file transfer path.
func (w imageCacheWriter) ReadFrom(src io.Reader) (int64, error) {
	n, err := io.Copy(w.ResponseWriter, src)
	if err != nil {
		log.Print("Could not write cached response")
	}
	return n, err
}

type imageProcessingResult struct {
	statusCode int
}

func (backend *Backend) acquireProcessingSlot(ctx context.Context) error {
	select {
	case backend.processingSlots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (backend *Backend) releaseProcessingSlot() {
	<-backend.processingSlots
}

func positiveOrDefault(value, defaultValue int) int {
	if value > 0 {
		return value
	}
	return defaultValue
}

func writeHTTPError(w http.ResponseWriter, statusCode int) {
	w.Header().Del("Content-Type")
	http.Error(w, http.StatusText(statusCode), statusCode)
}
