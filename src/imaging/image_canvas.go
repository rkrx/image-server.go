package imaging

import (
	"image"
	"image/color"
	"image/draw"

	xdraw "golang.org/x/image/draw"
)

type Positioning uint

const (
	Start  Positioning = 0
	Center Positioning = 1
	End    Positioning = 2
)

func SetCanvas(img image.Image, width, height int, horizontalPos, verticalPos Positioning, backgroundColor color.Color) image.Image {
	imgDx := img.Bounds().Dx()
	imgDy := img.Bounds().Dy()
	if imgDx == width && imgDy == height {
		return img
	}
	if width == 0 {
		width = imgDx
	}
	if height == 0 {
		height = imgDy
	}
	rect := image.Rect(0, 0, width, height)
	newImg := image.NewRGBA(rect)
	draw.Draw(newImg, newImg.Rect, &image.Uniform{C: backgroundColor}, image.Point{}, draw.Src)
	sb := img.Bounds()
	db := newImg.Bounds()
	x := alignValue(db.Dx(), imgDx, horizontalPos)
	y := alignValue(db.Dy(), imgDy, verticalPos)
	xdraw.Copy(newImg, image.Point{X: x, Y: y}, img, sb, draw.Over, nil)
	return newImg
}

func alignValue(canvasLength, objectLength int, pos Positioning) int {
	if pos == Start {
		return 0
	} else if pos == End {
		return canvasLength - objectLength
	}
	return canvasLength/2 - objectLength/2
}
