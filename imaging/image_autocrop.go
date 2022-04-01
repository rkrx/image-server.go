package imaging

import (
	"golang.org/x/image/draw"
	"image"
	"image/color"
	"math"
)

func Autocrop(img image.Image, tolerancePercent, borderPercent float64) image.Image {
	measures := getAutocropMeasures(img, tolerancePercent, borderPercent)
	newImg := image.NewRGBA(measures.target.Dim())
	draw.Draw(newImg, measures.target.Dim(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Copy(newImg, measures.target.Point(), img, measures.source.Rect(), draw.Over, nil)
	return newImg
}

func getAutocropMeasures(img image.Image, tolerancePercent, borderPercent float64) AutocropMeasures {
	absTolerance := 0xFFFF - uint32(math.Round(float64(0xFFFF)/100.0*tolerancePercent))

	rect := GetInnerObjectBounds(img, absTolerance)
	borderAbs := int(math.Round(math.Max(float64(rect.Dx())/100*borderPercent, float64(rect.Dy())/100*borderPercent)))

	srcX, offsetX := getCoords(rect.Min.X, borderAbs)
	srcY, offsetY := getCoords(rect.Min.Y, borderAbs)

	X1 := rect.Max.X + borderAbs*2
	if X1 > img.Bounds().Max.X {
		X1 = img.Bounds().Max.X
	}

	Y1 := rect.Bounds().Max.Y + borderAbs*2
	if Y1 > img.Bounds().Max.Y {
		Y1 = img.Bounds().Max.Y
	}

	return AutocropMeasures{
		source: AutocropCanvasDimensions{
			x: srcX,
			y: srcY,
			w: X1 - rect.Min.X,
			h: Y1 - rect.Min.Y,
		},
		target: AutocropCanvasDimensions{
			x: offsetX,
			y: offsetY,
			w: rect.Dx() + borderAbs*2,
			h: rect.Dy() + borderAbs*2,
		},
	}
}

func getCoords(p, b int) (newP, offset int) {
	if p-b < 0 {
		return 0, -(p - b)
	}
	return p - b, 0
}

type AutocropMeasures struct {
	source AutocropCanvasDimensions
	target AutocropCanvasDimensions
}

type AutocropCanvasDimensions struct {
	x int
	y int
	w int
	h int
}

func (dim *AutocropCanvasDimensions) Dim() image.Rectangle {
	return image.Rect(0, 0, dim.w, dim.h)
}

func (dim *AutocropCanvasDimensions) Rect() image.Rectangle {
	return image.Rect(dim.x, dim.y, dim.x+dim.w, dim.y+dim.h)
}

func (dim *AutocropCanvasDimensions) Point() image.Point {
	return image.Point{X: dim.x, Y: dim.y}
}
