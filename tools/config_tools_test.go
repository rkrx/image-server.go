package tools

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	conf, err := LoadConfig("test_assets/config.toml")
	assert.Nil(t, err)
	assert.True(t, conf.LoggingEnabled)

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
	assert.Equal(t, Profile{Name: "250px", Width: 250, Height: 250, Sharpen: 0.6, Autocrop: autocrop{Tolerance: 15, Border: 3}}, profile)
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
