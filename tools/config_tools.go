package tools

import (
	"github.com/BurntSushi/toml"
)

type autocrop struct {
	Tolerance uint32
	Border    float32
}

type profile struct {
	Name     string
	Width    uint32
	Height   uint32
	Sharpen  float32
	Autocrop autocrop
}

type ServerType struct {
	ListenAddr      string
	OriginalBackend string
}

type Config struct {
	LoggingEnabled bool       `toml:"enableLogging"`
	Server         ServerType `toml:"server"`
	Profiles       []profile  `toml:"profile"`
}

func LoadConfig(filename string) (Config, error) {
	var conf Config
	_, err := toml.DecodeFile(filename, &conf)
	return conf, err
}
