package server

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/chai2010/webp"
	"image"
	"image-server/src/imaging"
	"image-server/src/tools"
	"image/color"
	"image/gif"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Backend struct {
	Config tools.Config
}

func (backend Backend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("RequestURI %s", r.RequestURI)
	w.Header().Add("Cache-Control", "no-cache")

	profile := tools.Profile{Width: 0, Height: 0}

	if r.URL.Query().Has("profile") {
		profileName := r.URL.Query().Get("profile")
		var err error
		if profile, err = backend.Config.GetProfile(profileName); err != nil {
			w.WriteHeader(404)
			return
		}
	}

	if err := applyUrlParameters(profile, r.RequestURI, r.URL.Query()); err != nil {
		w.WriteHeader(403)
		return
	}

	if r.URL.Query().Has("type") {
		fileType := strings.ToLower(r.URL.Query().Get("type"))
		profile.ApplyFileType(fileType)
	}

	origFilename, cachePath := "", ""
	if backend.Config.HashCacheFilenames {
		origFilename = tools.Md5(fmt.Sprintf("%s/%s", profile.GetDirectoryHash(), r.URL.Path))
		cachePath = fmt.Sprintf("var/cache/images/%s.%s", origFilename, profile.Type)
	} else {
		origFilename = tools.Md5(r.URL.Path)
		cachePath = fmt.Sprintf("var/cache/images/%s/%s.%s", profile.GetDirectoryHash(), origFilename, profile.Type)
	}

	if success, groups := tools.RegexMatch(r.URL.Path, "\\.(jpe?g|gif|png|webp)$"); success {
		origFilename = fmt.Sprintf("%s.%s", origFilename, groups[1])
	}

	if !tools.FileExists(cachePath) {
		log.Printf("💢 File %s is not cached", cachePath)
		statusCode, err := backend.getFinalImageBytes(r.URL.Path, origFilename, cachePath, profile, func(img image.Image) image.Image {
			if profile.Width < 1 && profile.Height < 1 {
				profile.Width = img.Bounds().Dx()
				profile.Height = img.Bounds().Dy()
			}
			img = imaging.Autocrop(img, profile.Autocrop.Tolerance, profile.Autocrop.Border)
			img = imaging.ResizeProportional(img, profile.Width, profile.Height)
			img = imaging.Sharpen(img, float64(profile.Sharpen))
			img = imaging.SetCanvas(img, profile.Width, profile.Height, imaging.Center, imaging.Center, color.White)
			return img
		})

		if err != nil {
			w.WriteHeader(statusCode)
			return
		}
	} else {
		log.Printf("✅ File %s is cached", cachePath)
	}

	imgBytes, err := os.ReadFile(cachePath)
	if err != nil {
		log.Printf("Could not read file %s: %s", cachePath, err)
		w.WriteHeader(500)
		return
	}

	_, err = w.Write(imgBytes)
	if err != nil {
		log.Println(err)
		return
	}
}

func applyUrlParameters(profile tools.Profile, requestUri string, values url.Values) error {
	var tryApplyUrlParameter = func(key string, fn func(value string) bool) func() bool {
		return func() bool {
			if values.Has(key) {
				value := values.Get(key)
				return fn(value)
			}
			return false
		}
	}

	params := []func() bool{
		tryApplyUrlParameter("width", func(value string) bool {
			profile.Width = tools.ToIntOr(value, profile.Width)
			return true
		}),
		tryApplyUrlParameter("height", func(value string) bool {
			profile.Height = tools.ToIntOr(value, profile.Height)
			return true
		}),
		tryApplyUrlParameter("sharpen", func(value string) bool {
			profile.Sharpen = tools.ToFloat32Or(value, profile.Sharpen)
			return true
		}),
		tryApplyUrlParameter("autocrop", func(value string) bool {
			if success, groups := tools.RegexMatch(value, "^(\\d+)(?:,(\\d+))?$"); success {
				tolerance := tools.ToFloat32Or(groups[1], -1)
				border := tools.ToFloat32Or(groups[2], 0)
				profile.Autocrop.Tolerance = tolerance
				profile.Autocrop.Border = border
			}
			return true
		}),
	}

	parameterFound := false
	for _, fn := range params {
		if fn() {
			parameterFound = true
		}
	}

	if parameterFound && !extractHashFromRequestUri(requestUri, values) {
		return errors.New("hash value failure")
	}

	return nil
}

func extractHashFromRequestUri(requestUri string, values url.Values) bool {
	if values.Has("hash") {
		var hash string
		success, groups := tools.RegexMatch(requestUri, "\\?.*?&hash=([a-f0-9]+)(&|$)")
		if success {
			hash = groups[1]

			requestUriWoHash := tools.RegexReplace(requestUri, "(\\?.*?)(&hash=(?:[a-f0-9]+))(&|$)", "$1$3")
			log.Printf("RequestURI w/o hash %s", requestUriWoHash)
			if tools.Md5(requestUriWoHash) == hash {
				return true
			}
		}
	}
	return false
}

func (backend Backend) getFinalImageBytes(requestPath, filename, cachePath string, profile tools.Profile, fn func(image.Image) image.Image) (int, error) {
	dir := filepath.Dir(cachePath)
	os.MkdirAll(dir, os.ModePerm)

	cachedOriginalPath := fmt.Sprintf("var/cache/originals/%s", filename)

	if !tools.FileExists(cachedOriginalPath) {
		imageContent, statusCode, err := backend.FetchImage(requestPath)
		if err != nil {
			log.Printf("Error: %s (ResponseCode = %d; RequestPath = %s; Backend = %s)", err, statusCode, requestPath, profile.Backend)
			return statusCode, err
		}

		os.WriteFile(cachedOriginalPath, imageContent, 0666)
	}

	imageContent, err := os.ReadFile(cachedOriginalPath)
	if err != nil {
		log.Printf("Cache file not readable: %s", err)
		return 500, err
	}

	reader := bytes.NewReader(imageContent)
	img, errorStr, err := image.Decode(reader)
	if err != nil {
		log.Printf("Could not decode image data: %s (%s)", err, errorStr)
		return 500, err
	}

	img = fn(img)

	switch profile.Type {
	case tools.Jpeg:
		if err := imaging.SaveToJpegFile(img, cachePath, backend.Config.JpegDefaults.Quality); err != nil {
			return 500, err
		}
	case tools.WebP:
		quality := float32(backend.Config.WebpDefaults.Quality)
		lossless := backend.Config.WebpDefaults.Lossless
		if err := webp.Save(cachePath, img, &webp.Options{Quality: quality, Lossless: lossless}); err != nil {
			return 500, err
		}
	case tools.Png:
		if err := imaging.SaveToPngFile(img, cachePath); err != nil {
			return 500, err
		}
	case tools.Gif:
		if err := imaging.SaveToGifFile(img, cachePath, &gif.Options{}); err != nil {
			return 500, err
		}
	}

	return 500, errors.New(fmt.Sprintf("no encoder for filetype %s implemented", profile.Type))
}

func parseRequest(requestPath string) (width int, height int) {
	if success, groups := tools.RegexMatch(requestPath, "^/(\\d+)-(\\d+)/(.*?)$"); success {
		width = tools.ToIntOr(groups[1], width)
		height = tools.ToIntOr(groups[2], height)
	} else {
		width = 0
		height = 0
	}
	return
}

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
