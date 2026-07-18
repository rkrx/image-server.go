package imaging

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResizeUsesRequestedDimensions(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 20))

	resized := Resize(img, 5, 7)

	assert.Equal(t, 5, resized.Bounds().Dx())
	assert.Equal(t, 7, resized.Bounds().Dy())
}

func TestSaveToAvifFile(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 6))
	for y := 0; y < 6; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 20), G: uint8(y * 30), B: 120, A: 255})
		}
	}
	filename := filepath.Join(t.TempDir(), "test.avif")

	assert.NoError(t, SaveToAvifFile(img, filename, 60, 80, 6))

	file, err := os.Open(filename)
	assert.NoError(t, err)
	defer file.Close()
	decoded, format, err := image.Decode(file)
	assert.NoError(t, err)
	assert.Equal(t, "avif", format)
	assert.Equal(t, img.Bounds(), decoded.Bounds())
	fileInfo, err := os.Stat(filename)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())
}
