package imaging

import (
	"bytes"
	"github.com/chai2010/webp"
	"image"
	"image/jpeg"
	"log"
	"os"
)

func Width(img image.Image) int {
	return img.Bounds().Dx()
}

func Height(img image.Image) int {
	return img.Bounds().Dy()
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
