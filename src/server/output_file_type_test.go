package server

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"image-server/src/tools"
)

func TestApplyOutputFileTypeUsesExplicitType(t *testing.T) {
	for _, fileType := range []tools.FileType{tools.Jpeg, tools.WebP, tools.Avif} {
		request := httptest.NewRequest("GET", "/image.jpg?type="+string(fileType), nil)
		request.Header.Set("Accept", "image/avif,image/webp")
		profile := tools.Profile{}

		negotiated, err := applyOutputFileType(&profile, request)

		assert.NoError(t, err)
		assert.False(t, negotiated)
		assert.Equal(t, fileType, profile.Type)
	}
}

func TestApplyOutputFileTypeNegotiatesAcceptHeader(t *testing.T) {
	testCases := []struct {
		name           string
		accept         string
		expectedFormat tools.FileType
	}{
		{name: "prefers avif", accept: "image/avif,image/webp,*/*;q=0.8", expectedFormat: tools.Avif},
		{name: "uses webp", accept: "image/webp,*/*;q=0.8", expectedFormat: tools.WebP},
		{name: "honors quality", accept: "image/avif;q=0.5,image/webp;q=0.9", expectedFormat: tools.WebP},
		{name: "ignores disabled avif", accept: "image/avif;q=0,image/webp", expectedFormat: tools.WebP},
		{name: "falls back to jpeg", accept: "image/png,*/*", expectedFormat: tools.Jpeg},
		{name: "falls back without accept", accept: "", expectedFormat: tools.Jpeg},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/image.jpg", nil)
			request.Header.Set("Accept", testCase.accept)
			profile := tools.Profile{Type: tools.Png}

			negotiated, err := applyOutputFileType(&profile, request)

			assert.NoError(t, err)
			assert.True(t, negotiated)
			assert.Equal(t, testCase.expectedFormat, profile.Type)
		})
	}
}
