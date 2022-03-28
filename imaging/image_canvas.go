package imaging

import (
	draw2 "golang.org/x/image/draw"
	"image"
	"image/color"
	"image/draw"
)

type Positioning uint

const (
	Start  Positioning = 0
	Center             = 1
	End                = 2
)

func SetCanvas(img image.Image, width, height int, horizontalPos, verticalPos Positioning, backgroundColor color.Color) image.Image {
	rect := image.Rect(0, 0, width, height)
	newImg := image.NewRGBA(rect)
	draw.Draw(newImg, newImg.Rect, &image.Uniform{C: backgroundColor}, image.Point{}, draw.Src)
	sb := img.Bounds()
	db := newImg.Bounds()
	x := alignValue(db.Dx(), sb.Dx(), horizontalPos)
	y := alignValue(db.Dy(), sb.Dy(), verticalPos)
	draw2.Copy(newImg, image.Point{X: x, Y: y}, img, sb, draw.Src, nil)
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
