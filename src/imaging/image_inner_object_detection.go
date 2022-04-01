package imaging

import (
	"golang.org/x/image/draw"
	copy "golang.org/x/image/draw"
	"image"
	"image/color"
)

func GetInnerObjectBounds(img image.Image, toleranceUpper uint32) image.Rectangle {
	w := Width(img)
	h := Height(img)

	newImg := image.NewNRGBA(img.Bounds())
	draw.Draw(newImg, newImg.Rect, &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	copy.Copy(newImg, image.Point{X: 0, Y: 0}, img, img.Bounds(), draw.Over, nil)

	t := getBorderBounds(newImg, 0, w-1, 1, 0, h-1, 1, toleranceUpper, false)
	r := getBorderBounds(newImg, 0, h-1, 1, w-1, 0, -1, toleranceUpper, true)
	b := getBorderBounds(newImg, 0, w-1, 1, h-1, 0, -1, toleranceUpper, false)
	l := getBorderBounds(newImg, 0, h-1, 1, 0, w-1, 1, toleranceUpper, true)

	return image.Rect(l+1, t+1, r, b)
}

func getBorderBounds(img *image.NRGBA, startX int, endX int, incX int, startY int, endY int, incY int, toleranceUpper uint32, transpose bool) int {
	var r, g, b uint32
	var x, y int

	for y = startY; y != endY; y += incY {
		for x = startX; x != endX; x += incX {
			if transpose {
				r, g, b, _ = img.At(y, x).RGBA()
			} else {
				r, g, b, _ = img.At(x, y).RGBA()
			}

			if r < toleranceUpper || g < toleranceUpper || b < toleranceUpper {
				return y - incY
			}
		}
	}

	return endY
}
