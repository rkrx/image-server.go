package imaging

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetAutocropMeasures(t *testing.T) {
	img, _ := LoadImageFromFile("test_assets/test32.jpeg")

	measures := getAutocropMeasures(img, 15, 0)
	assert.Equal(t, 6, measures.source.x)
	assert.Equal(t, 9, measures.source.y)
	assert.Equal(t, 23, measures.source.w)
	assert.Equal(t, 17, measures.source.h)
	assert.Equal(t, 0, measures.target.x)
	assert.Equal(t, 0, measures.target.y)
	assert.Equal(t, 23, measures.target.w)
	assert.Equal(t, 17, measures.target.h)

	measures = getAutocropMeasures(img, 15, 3)
	assert.Equal(t, 5, measures.source.x)
	assert.Equal(t, 8, measures.source.y)
	assert.Equal(t, 25, measures.source.w)
	assert.Equal(t, 19, measures.source.h)
	assert.Equal(t, 0, measures.target.x)
	assert.Equal(t, 0, measures.target.y)
	assert.Equal(t, 25, measures.target.w)
	assert.Equal(t, 19, measures.target.h)

	measures = getAutocropMeasures(img, 15, 100)
	assert.Equal(t, 0, measures.source.x)
	assert.Equal(t, 0, measures.source.y)
	assert.Equal(t, 26, measures.source.w)
	assert.Equal(t, 23, measures.source.h)
	assert.Equal(t, 17, measures.target.x)
	assert.Equal(t, 14, measures.target.y)
	assert.Equal(t, 69, measures.target.w)
	assert.Equal(t, 63, measures.target.h)

	img, _ = LoadImageFromFile("test_assets/test_full32.jpeg")

	measures = getAutocropMeasures(img, 15, 0)
	assert.Equal(t, 0, measures.source.x)
	assert.Equal(t, 0, measures.source.y)
	assert.Equal(t, 32, measures.source.w)
	assert.Equal(t, 32, measures.source.h)
	assert.Equal(t, 0, measures.target.x)
	assert.Equal(t, 0, measures.target.y)
	assert.Equal(t, 32, measures.target.w)
	assert.Equal(t, 32, measures.target.h)

	measures = getAutocropMeasures(img, 15, 3)
	assert.Equal(t, 0, measures.source.x)
	assert.Equal(t, 0, measures.source.y)
	assert.Equal(t, 32, measures.source.w)
	assert.Equal(t, 32, measures.source.h)
	assert.Equal(t, 1, measures.target.x)
	assert.Equal(t, 1, measures.target.y)
	assert.Equal(t, 34, measures.target.w)
	assert.Equal(t, 34, measures.target.h)

	measures = getAutocropMeasures(img, 15, 100)
	assert.Equal(t, 0, measures.source.x)
	assert.Equal(t, 0, measures.source.y)
	assert.Equal(t, 32, measures.source.w)
	assert.Equal(t, 32, measures.source.h)
	assert.Equal(t, 32, measures.target.x)
	assert.Equal(t, 32, measures.target.y)
	assert.Equal(t, 96, measures.target.w)
	assert.Equal(t, 96, measures.target.h)
}

func TestGetCoords(t *testing.T) {
	x, offset := getCoords(100, 30)
	assert.Equal(t, 70, x)
	assert.Equal(t, 0, offset)

	x, offset = getCoords(20, 30)
	assert.Equal(t, 0, x)
	assert.Equal(t, 10, offset)
}
