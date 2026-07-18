package server

import (
	"bytes"
	"context"
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

func (backend *Backend) getFinalImageBytes(ctx context.Context, requestPath, filename, cachePath string, profile tools.Profile, fn func(image.Image) (image.Image, error)) (int, error) {
	log.Print("Processing image request")

	dir := filepath.Dir(cachePath)
	if !tools.FileExists(dir) {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return 500, err
		}
	}
	// #nosec G302 -- cache directories require execute permission for traversal.
	if err := os.Chmod(dir, 0750); err != nil {
		return 500, err
	}

	cachedOriginalPath := filepath.Join(profile.ImageCacheDir, filename)

	if !tools.FileExists(cachedOriginalPath) {
		imageContent, statusCode, err := backend.FetchImage(ctx, requestPath)
		if err != nil {
			log.Print("Origin request failed")
			return statusCode, err
		}

		if err := os.WriteFile(cachedOriginalPath, imageContent, 0600); err != nil {
			return 500, err
		}
	}
	if err := os.Chmod(cachedOriginalPath, 0600); err != nil {
		return 500, err
	}

	imageContent, err := os.ReadFile(cachedOriginalPath)
	if err != nil {
		log.Print("Cache file not readable")
		return 500, err
	}

	imageConfig, _, err := image.DecodeConfig(bytes.NewReader(imageContent))
	if err != nil {
		return 415, fmt.Errorf("could not decode image configuration: %w", err)
	}
	if err := backend.Config.ValidateSourceImageDimensions(imageConfig.Width, imageConfig.Height); err != nil {
		return 413, err
	}

	img, _, err := image.Decode(bytes.NewReader(imageContent))
	if err != nil {
		log.Print("Could not decode image data")
		return 415, err
	}

	img, err = fn(img)
	if err != nil {
		return 413, err
	}
	if err := backend.Config.ValidateOutputImageDimensions(img.Bounds().Dx(), img.Bounds().Dy()); err != nil {
		return 413, err
	}

	switch profile.Type {
	case tools.Jpeg:
		if err := imaging.SaveToJpegFile(img, cachePath, backend.Config.JpegDefaults.Quality); err != nil {
			return 500, err
		}
	case tools.WebP:
		quality := float32(backend.Config.WebpDefaults.Quality)
		lossless := backend.Config.WebpDefaults.Lossless
		var output bytes.Buffer
		if err := webp.Encode(&output, img, &webp.Options{Quality: quality, Lossless: lossless}); err != nil {
			return 500, err
		}
		if err := os.WriteFile(cachePath, output.Bytes(), 0600); err != nil {
			return 500, err
		}
	case tools.Avif:
		if err := imaging.SaveToAvifFile(
			img,
			cachePath,
			backend.Config.AvifDefaults.Quality,
			backend.Config.AvifDefaults.AlphaQuality,
			backend.Config.AvifDefaults.Speed,
		); err != nil {
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
		return 500, fmt.Errorf("no encoder for filetype %s implemented", profile.Type)
	}

	return 200, nil
}
