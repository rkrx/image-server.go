package imaging

import (
	"image"
	"math"
)

func GetInnerObjectBounds(img image.Image, toleranceMin uint32, toleranceMax uint32) image.Rectangle {
	w := Width(img)
	h := Height(img)

	t := getBorderBounds(img, 0, w-1, 1, 0, h-1, 1, toleranceMin, toleranceMax, false)
	r := getBorderBounds(img, 0, h-1, 1, w-1, 0, -1, toleranceMin, toleranceMax, true)
	b := getBorderBounds(img, 0, w-1, 1, h-1, 0, -1, toleranceMin, toleranceMax, false)
	l := getBorderBounds(img, 0, h-1, 1, 0, w-1, 1, toleranceMin, toleranceMax, true)

	return image.Rect(l+1, t+1, r, b)
}

func getBorderBounds(img image.Image, startX int, endX int, incX int, startY int, endY int, incY int, toleranceMin uint32, toleranceMax uint32, transpose bool) int {
	var r, g, b, a uint32
	var x, y int
	alphaTolerance := uint32(math.Round(0xFFFF * 0.05)) // Activation when alpha is higher than 5%

	for y = startY; y != endY; y += incY {
		for x = startX; x != endX; x += incX {
			if transpose {
				r, g, b, a = img.At(y, x).RGBA()
			} else {
				r, g, b, a = img.At(x, y).RGBA()
			}

			if (r < toleranceMin || r > toleranceMax) && (g < toleranceMin || g > toleranceMax) && (b < toleranceMin || b > toleranceMax) && a > alphaTolerance {
				return y - incY
			}
		}
	}

	return endY
}
