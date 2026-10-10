package server

import (
	"image"
	"image-server/src/imaging"
	"image-server/src/tools"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransformImagePreservesSingleDimensionStretchSizing(t *testing.T) {
	backend := NewBackend(tools.Config{})
	source := image.NewNRGBA(image.Rect(0, 0, 17, 11))
	for _, size := range [][2]int{{3, 0}, {0, 4}, {0, 5}} {
		profile := backend.Config.GetDefaultProfile()
		profile.Mode, profile.Width, profile.Height = "stretch", size[0], size[1]
		want := imaging.Resize(source, size[0], size[1])
		got, err := backend.transformImage(source, profile, true, nil)
		require.NoError(t, err)
		require.Equal(t, want.Bounds(), got.Bounds(), "size %v must retain existing stretch rounding", size)
	}
}
