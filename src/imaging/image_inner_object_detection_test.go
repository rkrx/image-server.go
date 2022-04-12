package imaging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetInnerObjectBounds(t *testing.T) {
	img, _ := LoadImageFromFile("test_assets/test_full32.jpeg")
	m := GetInnerObjectBounds(img, 0xFFFF-0xFFFF/100*15)
	assert.Equal(t, 0, m.Min.X)
	assert.Equal(t, 0, m.Min.Y)
	assert.Equal(t, 32, m.Max.X)
	assert.Equal(t, 32, m.Max.Y)

	img, _ = LoadImageFromFile("test_assets/test32.jpeg")
	m = GetInnerObjectBounds(img, 0xFFFF-0xFFFF/100*15)
	assert.Equal(t, 6, m.Min.X)
	assert.Equal(t, 9, m.Min.Y)
	assert.Equal(t, 29, m.Max.X)
	assert.Equal(t, 26, m.Max.Y)
}

func TestGetInnerObjectBounds_Alpha(t *testing.T) {
	img, _ := LoadImageFromFile("test_assets/test32.png")
	m := GetInnerObjectBounds(img, 0xFFFF-0xFFFF/100*15)
	assert.Equal(t, 6, m.Min.X)
	assert.Equal(t, 9, m.Min.Y)
	assert.Equal(t, 29, m.Max.X)
	assert.Equal(t, 27, m.Max.Y)
}
