package imaging

import (
	"bytes"
	"github.com/chai2010/webp"
	im "github.com/disintegration/imaging"
	"github.com/nfnt/resize"
	exifremove "github.com/scottleedavis/go-exif-remove"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	_ "image/png"
	"log"
	"os"
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
	if imgW != width && imgH != height {
		return resize.Resize(uint(imgW), uint(imgH), img, resize.Lanczos3)
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
		return os.WriteFile(filepath, noExifBytes, 0666)
	}
	return os.WriteFile(filepath, imBytes, 0666)
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
		return os.WriteFile(filepath, noExifBytes, 0666)
	}
	return os.WriteFile(filepath, imBytes, 0666)
}

func SaveToGifFile(img image.Image, filepath string, options *gif.Options) error {
	var buf = bytes.Buffer{}
	err := gif.Encode(&buf, img, options)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath, buf.Bytes(), 0666)
}

func Img2WebpBuffer(buf *bytes.Buffer, img image.Image, quality float32) error {
	if err := webp.Encode(buf, img, &webp.Options{Quality: quality}); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func Img2JpegBytes(img image.Image, quality int) ([]byte, error) {
	buf := bytes.Buffer{}
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
	return buf.Bytes(), err
}

func Img2JpegBuffer(buf *bytes.Buffer, img image.Image, quality int) error {
	return jpeg.Encode(buf, img, &jpeg.Options{Quality: quality})
}
