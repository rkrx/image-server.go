package server

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/chai2010/webp"
	"image"
	"image-server/src/imaging"
	"image-server/src/tools"
	"image/gif"
	"log"
	"os"
	"path/filepath"
)

func (backend Backend) getFinalImageBytes(requestPath, filename, cachePath string, profile tools.Profile, fn func(image.Image) image.Image) (int, error) {
	log.Printf("Request: %s", requestPath)

	dir := filepath.Dir(cachePath)
	if !tools.FileExists(dir) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return 500, err
		}
	}

	cachedOriginalPath := fmt.Sprintf("%s/%s", profile.ImageOriginalDir, filename)

	if !tools.FileExists(cachedOriginalPath) {
		imageContent, statusCode, err := backend.FetchImage(requestPath)
		if err != nil {
			log.Printf("Error: %s (ResponseCode = %d; RequestPath = %s; Backend = %s)", err, statusCode, requestPath, profile.Backend)
			return statusCode, err
		}

		if err := os.WriteFile(cachedOriginalPath, imageContent, 0666); err != nil {
			return 500, err
		}
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
	default:
		return 500, errors.New(fmt.Sprintf("no encoder for filetype %s implemented", profile.Type))
	}

	return 200, nil
}
