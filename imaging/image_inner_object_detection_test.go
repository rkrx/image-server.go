package imaging

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetInnerObjectBounds(t *testing.T) {
	img, _ := LoadImageFromFile("assets/test_full32.jpeg")
	m := GetInnerObjectBounds(img, 0xFB04, 0xFFFF)
	assert.Equal(t, 0, m.Min.X)
	assert.Equal(t, 0, m.Min.Y)
	assert.Equal(t, 32, m.Max.X)
	assert.Equal(t, 32, m.Max.Y)

	img, _ = LoadImageFromFile("assets/test32.jpeg")
	m = GetInnerObjectBounds(img, 0xF335, 0xFFFF)
	assert.Equal(t, 6, m.Min.X)
	assert.Equal(t, 9, m.Min.Y)
	assert.Equal(t, 29, m.Max.X)
	assert.Equal(t, 26, m.Max.Y)
}
