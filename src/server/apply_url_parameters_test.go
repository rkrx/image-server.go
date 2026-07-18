package server

import (
	"github.com/stretchr/testify/assert"
	"image-server/src/tools"
	"net/url"
	"testing"
)

func Test_applyUrlParameters(t *testing.T) {
	var err error
	var profile tools.Profile
	var requestUri string
	var parts *url.URL
	config := tools.Config{
		Secret:        "test",
		HashAlgorithm: "md5",
		HashLength:    8,
	}

	// With hash checking disabled
	profile = tools.Profile{}
	requestUri = "/image.jpg?width=400&height=500&autocrop=15,3&sharpen=.6&type=png"
	parts, err = url.ParseRequestURI(requestUri)
	assert.Nil(t, err)
	err = applyUrlParameters(&profile, config, requestUri, parts.Query(), true)
	assert.Nil(t, err)
	assert.Equal(t, "", profile.Name)
	assert.Equal(t, 400, profile.Width)
	assert.Equal(t, 500, profile.Height)
	assert.Equal(t, float32(0.6), profile.Sharpen)
	assert.Equal(t, float32(15.0), profile.Autocrop.Tolerance)
	assert.Equal(t, float32(3.0), profile.Autocrop.Border)
	//assert.Equal(t, tools.Png, profile.Type)

	// With hash checking enabled
	profile = tools.Profile{}
	requestUri = "/image.jpg?width=400&height=500&autocrop=15,3&sharpen=.6&type=png&hash=9acd605b"
	parts, err = url.ParseRequestURI(requestUri)
	assert.Nil(t, err)
	err = applyUrlParameters(&profile, config, requestUri, parts.Query(), false)
	assert.Nil(t, err)
	assert.Equal(t, "", profile.Name)
	assert.Equal(t, 400, profile.Width)
	assert.Equal(t, 500, profile.Height)
	assert.Equal(t, float32(0.6), profile.Sharpen)
	assert.Equal(t, float32(15.0), profile.Autocrop.Tolerance)
	assert.Equal(t, float32(3.0), profile.Autocrop.Border)
	//assert.Equal(t, tools.Png, profile.Type)
}

func Test_extractHashFromRequestUri_DefaultsToLegacyMd5Hash(t *testing.T) {
	requestUri := "/image.jpg?width=400&height=500&autocrop=15,3&sharpen=.6&type=png&hash=9acd605b"
	parts, err := url.ParseRequestURI(requestUri)
	assert.Nil(t, err)

	assert.True(t, extractHashFromRequestUri(requestUri, parts.Query(), tools.Config{Secret: "test"}))
}

func Test_extractHashFromRequestUri_UsesConfiguredSha256HashLength(t *testing.T) {
	config := tools.Config{
		Secret:        "test",
		HashAlgorithm: "sha256",
		HashLength:    12,
	}
	requestUriWithoutHash := "/image.jpg?width=400&height=500&autocrop=15,3&sharpen=.6"
	hash := generateRequestHash(requestUriWithoutHash, config)
	requestUri := requestUriWithoutHash + "&hash=" + hash
	parts, err := url.ParseRequestURI(requestUri)
	assert.Nil(t, err)

	assert.Len(t, hash, 12)
	assert.True(t, extractHashFromRequestUri(requestUri, parts.Query(), config))
}

func TestApplyUrlParametersRejectsUnsafeValues(t *testing.T) {
	config := tools.Config{
		MaxOutputImageWidth:  1000,
		MaxOutputImageHeight: 1000,
		MaxOutputImagePixels: 500000,
	}
	testCases := []string{
		"/image.jpg?width=-1",
		"/image.jpg?width=1001",
		"/image.jpg?width=1000&height=1000",
		"/image.jpg?mode=invalid",
		"/image.jpg?sharpen=NaN",
		"/image.jpg?sharpen=11",
		"/image.jpg?autocrop=101,0",
		"/image.jpg?autocrop=10,26",
	}

	for _, requestURI := range testCases {
		t.Run(requestURI, func(t *testing.T) {
			parts, err := url.ParseRequestURI(requestURI)
			assert.NoError(t, err)

			profile := tools.Profile{}
			assert.Error(t, applyUrlParameters(&profile, config, requestURI, parts.Query(), true))
		})
	}
}
