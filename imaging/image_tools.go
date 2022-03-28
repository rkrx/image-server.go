package imaging

import (
	"bytes"
	"github.com/chai2010/webp"
	im "github.com/disintegration/imaging"
	"github.com/nfnt/resize"
	"image"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
)

func Width(img image.Image) int {
	return img.Bounds().Dx()
}

func Height(img image.Image) int {
	return img.Bounds().Dy()
}

func ResizeProportional(img image.Image, width, height int) image.Image {
	imgW, imgH := Width(img), Height(img)
	w, h := CalcProportionalResize(imgW, imgH, width, height)
	return resize.Resize(uint(w), uint(h), img, resize.Lanczos3)
}

func Sharpen(img image.Image, sigma float64) *image.NRGBA {
	return im.Sharpen(img, sigma)
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

func Img2WebpBuffer(buf *bytes.Buffer, img image.Image, quality float32) error {
	if err := webp.Encode(buf, img, &webp.Options{Quality: quality}); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func Img2JpegBuffer(buf *bytes.Buffer, img image.Image, quality int) error {
	return jpeg.Encode(buf, img, &jpeg.Options{Quality: quality})
}
