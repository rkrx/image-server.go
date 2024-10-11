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
	secret := "test"

	// With hash checking disabled
	profile = tools.Profile{}
	requestUri = "/image.jpg?width=400&height=500&autocrop=15,3&sharpen=.6&type=png"
	parts, err = url.ParseRequestURI(requestUri)
	assert.Nil(t, err)
	err = applyUrlParameters(&profile, tools.Config{}, requestUri, parts.Query(), secret, true)
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
	err = applyUrlParameters(&profile, tools.Config{}, requestUri, parts.Query(), secret, false)
	assert.Nil(t, err)
	assert.Equal(t, "", profile.Name)
	assert.Equal(t, 400, profile.Width)
	assert.Equal(t, 500, profile.Height)
	assert.Equal(t, float32(0.6), profile.Sharpen)
	assert.Equal(t, float32(15.0), profile.Autocrop.Tolerance)
	assert.Equal(t, float32(3.0), profile.Autocrop.Border)
	//assert.Equal(t, tools.Png, profile.Type)
}
