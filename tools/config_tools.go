package tools

import (
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
)

type autocrop struct {
	Tolerance float32
	Border    float32
}

type FileType string

const (
	Jpeg FileType = "jpeg"
	WebP          = "webp"
	Png           = "png"
	Gif           = "gif"
)

type Profile struct {
	Name     string
	Backend  string
	Type     FileType
	Width    int
	Height   int
	Sharpen  float32
	Autocrop autocrop
}

func (profile Profile) GetDirectoryHash() string {
	var name = "dynamic"
	if profile.Name != "" {
		name = fmt.Sprintf("name=%s", profile.Name)
	}
	var backendHash = fmt.Sprintf("%x", md5.Sum([]byte(profile.Backend)))
	return fmt.Sprintf("backend-hash=%s;%s;%d-%d;sh=%.3f;ac=%.3f,%.3f",
		backendHash,
		name,
		profile.Width,
		profile.Height,
		profile.Sharpen,
		profile.Autocrop.Tolerance,
		profile.Autocrop.Border,
	)
}

type ServerType struct {
	ListenAddr      string
	OriginalBackend string
}

type Config struct {
	LoggingEnabled     bool       `toml:"enableLogging"`
	HashCacheFilenames bool       `toml:"hashCacheFilenames"`
	Server             ServerType `toml:"server"`
	Profiles           []Profile  `toml:"Profile"`
}

func (conf Config) GetProfile(name string) (Profile, error) {
	for _, profile := range conf.Profiles {
		if profile.Name == name {
			return profile, nil
		}
	}
	return Profile{}, errors.New("profile not found")
}

func LoadConfig(filename string) (Config, error) {
	var conf Config
	_, err := toml.DecodeFile(filename, &conf)
	return conf, err
}
