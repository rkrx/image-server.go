package server

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image-server/imaging"
	"image-server/tools"
	"image/color"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Backend struct {
	BaseUri string
	Logging bool
}

func (backend Backend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("RequestURI %s", r.RequestURI)
	w.Header().Add("Cache-Control", "no-cache")

	filename, width, height := parseRequestPath(r.RequestURI)

	cachePath := fmt.Sprintf("var/cache/images/%d-%d/%s", width, height, filename)

	if !tools.FileExists(cachePath) {
		log.Printf("💢 File %s is not cached", cachePath)
		statusCode, err := backend.getFinalImageBytes(cachePath, func(img image.Image) image.Image {
			if width == 0 && height == 0 {
				width = img.Bounds().Dx()
				height = img.Bounds().Dy()
			}
			img = imaging.Autocrop(img, 15, 3)
			img = imaging.ResizeProportional(img, width, height)
			img = imaging.Sharpen(img, .6)
			img = imaging.SetCanvas(img, width, height, imaging.Center, imaging.Center, color.White)
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

func (backend Backend) getFinalImageBytes(cachePath string, fn func(image.Image) image.Image) (int, error) {
	dir := filepath.Dir(cachePath)
	os.MkdirAll(dir, os.ModePerm)

	filename := filepath.Base(cachePath)
	cachedOriginalPath := fmt.Sprintf("var/cache/originals/%s", filename)

	if !tools.FileExists(cachedOriginalPath) {
		imageContent, statusCode, err := backend.FetchImage(fmt.Sprintf("/%s", filename))
		if err != nil {
			log.Printf("Error: %s (ResponseCode = %d)", err, statusCode)
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

	imgBytes, err := imaging.Img2JpegBytes(img, 100)
	if err != nil {
		log.Println(err)
		return 500, err
	}

	os.WriteFile(cachePath, imgBytes, 0666)

	return 200, nil
}

func parseRequestPath(requestPath string) (filename string, width int, height int) {
	if success, groups := tools.RegexMatch(requestPath, "^/(\\d+)-(\\d+)/(.*?)$"); success {
		width = tools.ToIntOr(groups[1], width)
		height = tools.ToIntOr(groups[2], height)
		filename = groups[3]
	} else {
		width = 0
		height = 0
		filename = strings.TrimLeft(requestPath, "/")
	}
	return
}

func (backend *Backend) FetchImage(path string) ([]byte, int, error) {
	fetchUrl := fmt.Sprintf("%s%s", backend.BaseUri, path)
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
