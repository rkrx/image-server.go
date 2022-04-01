package server

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/chai2010/webp"
	"image"
	imaging2 "image-server/pkg/imaging"
	tools2 "image-server/pkg/tools"
	"image/color"
	"image/gif"
	_ "image/gif"
	"image/png"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Backend struct {
	Config tools2.Config
}

func (backend Backend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("RequestURI %s", r.RequestURI)
	w.Header().Add("Cache-Control", "no-cache")

	profile := tools2.Profile{Width: 0, Height: 0}

	if r.URL.Query().Has("profile") {
		profileName := r.URL.Query().Get("profile")
		var err error
		profile, err = backend.Config.GetProfile(profileName)
		if err != nil {
			w.WriteHeader(404)
			return
		}
	}

	var handleParameter = func(key string, fn func(value string) bool) func() bool {
		return func() bool {
			if r.URL.Query().Has(key) {
				if !extractHashFromRequestUri(r.RequestURI, r.URL.Query()) {
					return false
				}

				value := r.URL.Query().Get(key)
				return fn(value)
			}
			return true
		}
	}

	params := []func() bool{
		handleParameter("width", func(value string) bool {
			profile.Width = tools2.ToIntOr(value, profile.Width)
			return true
		}),
		handleParameter("height", func(value string) bool {
			profile.Height = tools2.ToIntOr(value, profile.Height)
			return true
		}),
		handleParameter("sharpen", func(value string) bool {
			profile.Sharpen = tools2.ToFloat32Or(value, profile.Sharpen)
			return true
		}),
		handleParameter("autocrop", func(value string) bool {
			if success, groups := tools2.RegexMatch(value, "^(\\d+)(?:,(\\d+))?$"); success {
				tolerance := tools2.ToFloat32Or(groups[1], -1)
				border := tools2.ToFloat32Or(groups[2], 0)
				profile.Autocrop.Tolerance = tolerance
				profile.Autocrop.Border = border
			}
			return true
		}),
	}

	for _, fn := range params {
		if !fn() {
			w.WriteHeader(403)
			return
		}
	}

	if profile.Type == "" {
		if r.URL.Query().Has("type") {
			fileType := strings.ToLower(r.URL.Query().Get("type"))
			switch fileType {
			case "jpeg":
				profile.Type = tools2.Jpeg
				break
			case "webp":
				profile.Type = tools2.WebP
				break
			case "png":
				profile.Type = tools2.Png
				break
			case "gif":
				profile.Type = tools2.Gif
				break
			default:
				log.Printf("unknown file type: %s", fileType)
				return
			}
		}
	}

	origFilename, cachePath := "", ""
	if backend.Config.HashCacheFilenames {
		origFilename = tools2.Md5(fmt.Sprintf("%s/%s", profile.GetDirectoryHash(), r.URL.Path))
		cachePath = fmt.Sprintf("var/cache/images/%s.%s", origFilename, profile.Type)
	} else {
		origFilename = tools2.Md5(r.URL.Path)
		cachePath = fmt.Sprintf("var/cache/images/%s/%s.%s", profile.GetDirectoryHash(), origFilename, profile.Type)
	}

	if success, groups := tools2.RegexMatch(r.URL.Path, "\\.(jpe?g|gif|png|webp)$"); success {
		origFilename = fmt.Sprintf("%s.%s", origFilename, groups[1])
	}

	if !tools2.FileExists(cachePath) {
		log.Printf("💢 File %s is not cached", cachePath)
		statusCode, err := backend.getFinalImageBytes(r.URL.Path, origFilename, cachePath, profile, func(img image.Image) image.Image {
			if profile.Width < 1 && profile.Height < 1 {
				profile.Width = img.Bounds().Dx()
				profile.Height = img.Bounds().Dy()
			}
			img = imaging2.Autocrop(img, profile.Autocrop.Tolerance, profile.Autocrop.Border)
			img = imaging2.ResizeProportional(img, profile.Width, profile.Height)
			img = imaging2.Sharpen(img, float64(profile.Sharpen))
			img = imaging2.SetCanvas(img, profile.Width, profile.Height, imaging2.Center, imaging2.Center, color.White)
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

func extractHashFromRequestUri(requestUri string, values url.Values) bool {
	if values.Has("hash") {
		var hash string
		success, groups := tools2.RegexMatch(requestUri, "\\?.*?&hash=([a-f0-9]+)(&|$)")
		if success {
			hash = groups[1]

			requestUriWoHash := tools2.RegexReplace(requestUri, "(\\?.*?)(&hash=(?:[a-f0-9]+))(&|$)", "$1$3")
			log.Printf("RequestURI w/o hash %s", requestUriWoHash)
			if tools2.Md5(requestUriWoHash) == hash {
				return true
			}
		}
	}
	return false
}

func (backend Backend) getFinalImageBytes(requestPath, filename, cachePath string, profile tools2.Profile, fn func(image.Image) image.Image) (int, error) {
	dir := filepath.Dir(cachePath)
	os.MkdirAll(dir, os.ModePerm)

	cachedOriginalPath := fmt.Sprintf("var/cache/originals/%s", filename)

	if !tools2.FileExists(cachedOriginalPath) {
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

	if profile.Type == tools2.Jpeg {
		imgBytes, err := imaging2.Img2JpegBytes(img, 100)
		if err != nil {
			return 500, err
		}
		err = os.WriteFile(cachePath, imgBytes, 0666)
		if err != nil {
			return 500, err
		}
	} else if profile.Type == tools2.WebP {
		err := webp.Save(cachePath, img, &webp.Options{Quality: 90})
		if err != nil {
			return 500, err
		}
	} else if profile.Type == tools2.Png {
		var buf = bytes.Buffer{}
		err := png.Encode(&buf, img)
		if err != nil {
			return 500, err
		}
		err = os.WriteFile(cachePath, buf.Bytes(), 0666)
		if err != nil {
			return 500, err
		}
	} else if profile.Type == tools2.Gif {
		var buf = bytes.Buffer{}
		err := gif.Encode(&buf, img, &gif.Options{})
		if err != nil {
			return 500, err
		}
		err = os.WriteFile(cachePath, buf.Bytes(), 0666)
		if err != nil {
			return 500, err
		}
	}

	return 200, nil
}

func parseRequest(requestPath string) (width int, height int) {
	if success, groups := tools2.RegexMatch(requestPath, "^/(\\d+)-(\\d+)/(.*?)$"); success {
		width = tools2.ToIntOr(groups[1], width)
		height = tools2.ToIntOr(groups[2], height)
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
