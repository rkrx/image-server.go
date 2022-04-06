package tools

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	conf, err := LoadConfig("test_assets/config.toml")
	assert.Nil(t, err)
	assert.True(t, conf.LoggingEnabled)
	assert.True(t, conf.HashCacheFilenames)

	assert.Equal(t, 85, conf.JpegDefaults.Quality)
	assert.Equal(t, 80, conf.WebpDefaults.Quality)
	assert.True(t, conf.WebpDefaults.Lossless)

	assert.Equal(t, "0.0.0.0:8080", conf.Server.ListenAddr)
	assert.Equal(t, "https://127.0.0.1/original", conf.Server.OriginalBackend)

	assert.Equal(t, Profile{Name: "500px", Width: 500, Height: 500}, conf.Profiles[0])
	assert.Equal(t, Profile{Name: "250px", Width: 250, Height: 250, Sharpen: 0.6, Autocrop: autocrop{Tolerance: 15, Border: 3}}, conf.Profiles[1])
}

func TestConfig_GetProfile(t *testing.T) {
	conf, err := LoadConfig("test_assets/config.toml")
	assert.Nil(t, err)
	profile, err := conf.GetProfile("250px")
	assert.Nil(t, err)

	expectedProfile := Profile{
		Name:             "250px",
		ImageOriginalDir: "var/cache/originals",
		ImageCacheDir:    "var/cache/images",
		Type:             Jpeg,
		Width:            250,
		Height:           250,
		Sharpen:          0.6,
		Autocrop:         autocrop{Tolerance: 15, Border: 3},
	}

	assert.Equal(t, expectedProfile, profile)
}

func TestProfile_GetDirectoryHash(t *testing.T) {
	profile := Profile{
		Name:     "test",
		Backend:  "backend",
		Width:    250,
		Height:   250,
		Sharpen:  .6,
		Autocrop: autocrop{Tolerance: 15, Border: 3},
	}
	assert.Equal(t, "backend-hash=b43fdd98b1fd705ae4c3a10cf25aad8a;name=test;250-250;sh=0.600;ac=15.000,3.000", profile.GetDirectoryHash())
}

func TestProfile_ApplyFileType(t *testing.T) {
	var err error
	var profile Profile

	profile = Profile{}
	err = profile.ApplyFileType("")
	assert.Nil(t, err)
	assert.Equal(t, Jpeg, profile.Type)

	profile = Profile{}
	err = profile.ApplyFileType("jpg")
	assert.Nil(t, err)
	assert.Equal(t, Jpeg, profile.Type)

	profile = Profile{}
	err = profile.ApplyFileType("jpeg")
	assert.Nil(t, err)
	assert.Equal(t, Jpeg, profile.Type)

	profile = Profile{}
	err = profile.ApplyFileType("webp")
	assert.Nil(t, err)
	assert.Equal(t, WebP, profile.Type)

	profile = Profile{}
	err = profile.ApplyFileType("png")
	assert.Nil(t, err)
	assert.Equal(t, Png, profile.Type)

	profile = Profile{}
	err = profile.ApplyFileType("gif")
	assert.Nil(t, err)
	assert.Equal(t, Gif, profile.Type)

	profile = Profile{}
	err = profile.ApplyFileType("doc")
	assert.Error(t, err)
	assert.Equal(t, Jpeg, profile.Type)
}
