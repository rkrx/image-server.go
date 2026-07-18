package imaging

import (
	"image"
	"image/color"

	xdraw "golang.org/x/image/draw"
)

func GetInnerObjectBounds(img image.Image, toleranceUpper uint32) image.Rectangle {
	w := Width(img)
	h := Height(img)

	newImg := image.NewNRGBA(img.Bounds())
	xdraw.Draw(newImg, newImg.Rect, &image.Uniform{C: color.White}, image.Point{}, xdraw.Src)
	xdraw.Copy(newImg, image.Point{X: 0, Y: 0}, img, img.Bounds(), xdraw.Over, nil)

	t, r, b, l := make(chan int), make(chan int), make(chan int), make(chan int)

	go findBorderBounds(t, newImg, 0, w-1, 1, 0, h-1, 1, toleranceUpper, false)
	go findBorderBounds(r, newImg, 0, h-1, 1, w-1, 0, -1, toleranceUpper, true)
	go findBorderBounds(b, newImg, 0, w-1, 1, h-1, 0, -1, toleranceUpper, false)
	go findBorderBounds(l, newImg, 0, h-1, 1, 0, w-1, 1, toleranceUpper, true)

	return image.Rect(<-l+1, <-t+1, <-r, <-b)
}

func findBorderBounds(res chan int, img *image.NRGBA, startX int, endX int, incX int, startY int, endY int, incY int, toleranceUpper uint32, transpose bool) {
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
				res <- y - incY
				return
			}
		}
	}

	res <- endY
}
