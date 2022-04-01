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

	assert.Equal(t, profile{Name: "500px", Width: 500, Height: 500}, conf.Profiles[0])
	assert.Equal(t, profile{Name: "250px", Width: 250, Height: 250, Sharpen: 0.6, Autocrop: autocrop{Tolerance: 15, Border: 3}}, conf.Profiles[1])
}
