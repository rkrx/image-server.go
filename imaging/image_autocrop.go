package imaging

import (
	"golang.org/x/image/draw"
	"image"
	"math"
)

func Autocrop(img image.Image, tolerancePercentMin, tolerancePercentMax, borderPercent float64) image.Image {
	measures := getAutocropMeasures(img, tolerancePercentMin, tolerancePercentMax, borderPercent)
	newImg := image.NewRGBA(measures.target.Dim())
	draw.Copy(newImg, measures.target.Point(), img, measures.source.Rect(), draw.Src, nil)
	return newImg
}

func getAutocropMeasures(img image.Image, tolerancePercentMin, tolerancePercentMax, borderPercent float64) AutocropMeasures {
	absToleranceMin, absToleranceMax := getAbsToleranceValues(tolerancePercentMin, tolerancePercentMax)

	rect := GetInnerObjectBounds(img, absToleranceMin, absToleranceMax)
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

func getAbsToleranceValues(tolerancePercentMin, tolerancePercentMax float64) (uint32, uint32) {
	var absToleranceMin uint32 = 0
	var absToleranceMax uint32 = 0xFFFF

	if tolerancePercentMin > 0.0001 {
		absToleranceMin = uint32(math.Min(math.Max(math.Round(0xFFFF/100*tolerancePercentMin), 0), 0xFFFF))
	}

	if tolerancePercentMax < 99.999 {
		absToleranceMax = uint32(math.Min(math.Max(math.Round(0xFFFF/100*tolerancePercentMax), 0), 0xFFFF))
	}

	return absToleranceMin, absToleranceMax
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
