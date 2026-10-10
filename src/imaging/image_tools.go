package imaging

import (
	"bytes"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"

	im "github.com/disintegration/imaging"
	"github.com/nfnt/resize"
	exifremove "github.com/scottleedavis/go-exif-remove"
	avif "github.com/vegidio/avif-go"
)

type ImageResizeProjectionMode string

const (
	Contain ImageResizeProjectionMode = "contain"
	Cover   ImageResizeProjectionMode = "cover"
)

func Width(img image.Image) int {
	return img.Bounds().Dx()
}

func Height(img image.Image) int {
	return img.Bounds().Dy()
}

func Resize(img image.Image, width, height int) image.Image {
	imgW, imgH := Width(img), Height(img)
	if imgW != width || imgH != height {
		return resize.Resize(uint(width), uint(height), img, resize.Lanczos3)
	}
	return img
}

// ResizePixelated preserves hard pixel edges and transparency using nearest-neighbor sampling.
func ResizePixelated(img image.Image, width, height int) image.Image {
	if Width(img) != width || Height(img) != height {
		return resize.Resize(uint(width), uint(height), img, resize.NearestNeighbor)
	}
	return img
}

func ResizeProportional(img image.Image, width, height int, mode ImageResizeProjectionMode) image.Image {
	imgW, imgH := Width(img), Height(img)
	if imgW != width || imgH != height {
		if mode == Contain {
			w, h := CalcProportionalInnerMeasures(imgW, imgH, width, height)
			return resize.Resize(uint(w), uint(h), img, resize.Lanczos3)
		} else {
			w, h := CalcProportionalOuterMeasures(imgW, imgH, width, height)
			return resize.Resize(uint(w), uint(h), img, resize.Lanczos3)
		}
	}
	return img
}

func Sharpen(img image.Image, sigma float64) image.Image {
	if sigma > 0.000001 {
		return im.Sharpen(img, sigma)
	}
	return img
}

func LoadImageFromFile(filename string) (image.Image, error) {
	fimg, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fimg.Close()
	img, _, err := image.Decode(fimg)
	return img, err
}

func SaveToJpegFile(img image.Image, filepath string, quality int) error {
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
	if err != nil {
		return err
	}
	imBytes := buf.Bytes()
	noExifBytes, err := exifremove.Remove(imBytes)
	if err == nil {
		return os.WriteFile(filepath, noExifBytes, 0600)
	}
	return os.WriteFile(filepath, imBytes, 0600)
}

func SaveToPngFile(img image.Image, filepath string) error {
	var buf = bytes.Buffer{}
	err := png.Encode(&buf, img)
	if err != nil {
		return err
	}
	imBytes := buf.Bytes()
	noExifBytes, err := exifremove.Remove(imBytes)
	if err == nil {
		return os.WriteFile(filepath, noExifBytes, 0600)
	}
	return os.WriteFile(filepath, imBytes, 0600)
}

func SaveToGifFile(img image.Image, filepath string, options *gif.Options) error {
	var buf = bytes.Buffer{}
	err := gif.Encode(&buf, img, options)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath, buf.Bytes(), 0600)
}

func SaveToAvifFile(img image.Image, filepath string, quality, alphaQuality, speed int) error {
	var buf bytes.Buffer
	options := &avif.Options{
		ColorQuality: quality,
		AlphaQuality: alphaQuality,
		Speed:        speed,
	}
	if err := avif.Encode(&buf, img, options); err != nil {
		return err
	}
	return os.WriteFile(filepath, buf.Bytes(), 0600)
}
