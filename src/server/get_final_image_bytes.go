package server

import (
	"bytes"
	"context"
	"fmt"
	"github.com/chai2010/webp"
	"image"
	"image-server/src/imaging"
	"image-server/src/tools"
	"image/color"
	"image/draw"
	"image/gif"
	"os"
	"path/filepath"
)

func (backend *Backend) getFinalImageBytes(ctx context.Context, requestPath, filename, cachePath string, profile tools.Profile, budget *compositionBudget, virtualCanvas bool, fn func(image.Image) (image.Image, error)) (int, error) {
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

	var img image.Image
	if virtualCanvas {
		if err := backend.Config.ValidateOutputImageDimensions(profile.Width, profile.Height); err != nil {
			return 413, err
		}
		if err := budget.chargeRender(profile.Width, profile.Height); err != nil {
			return 413, err
		}
		canvas := image.NewRGBA(image.Rect(0, 0, profile.Width, profile.Height))
		draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
		img = canvas
	} else {
		var status int
		var err error
		img, status, err = backend.loadSourceImage(ctx, requestPath, filename, profile, budget)
		if err != nil {
			return status, err
		}
	}

	img, err := fn(img)
	if err != nil {
		return imageErrorStatus(err, 413), err
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
