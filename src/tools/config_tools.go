package tools

import (
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"log"
	"strings"
)

type autocrop struct {
	Tolerance float32
	Border    float32
}

type FileType string

const (
	Jpeg FileType = "jpeg"
	WebP FileType = "webp"
	Png  FileType = "png"
	Gif  FileType = "gif"
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

// ApplyFileType Sets Profile.Type to a valid file type based on string filetype. If fileType is not recognized or
// empty, the currently set file type will remain untoucher or - if empty - set to Jpeg.
func (profile *Profile) ApplyFileType(fileType string) error {
	switch strings.ToLower(fileType) {
	case "", "jpg", "jpeg":
		profile.Type = Jpeg
	case "webp":
		profile.Type = WebP
	case "png":
		profile.Type = Png
	case "gif":
		profile.Type = Gif
	default:
		if profile.Type == "" {
			profile.Type = Jpeg
		}
		return errors.New(fmt.Sprintf("unknown file type: %s", fileType))
	}
	return nil
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

type JpegDefaults struct {
	Quality int `toml:"quality"`
}

type WebpDefaults struct {
	Quality  int  `toml:"quality"`
	Lossless bool `toml:"lossless"`
}

type Config struct {
	LoggingEnabled     bool         `toml:"enableLogging"`
	Secret             string       `toml:"secret"`
	HashCacheFilenames bool         `toml:"hashCacheFilenames"`
	Server             ServerType   `toml:"server"`
	Profiles           []Profile    `toml:"profile"`
	JpegDefaults       JpegDefaults `toml:"jpegDefaults"`
	WebpDefaults       WebpDefaults `toml:"webpDefaults"`
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

	conf.LoggingEnabled = true
	conf.HashCacheFilenames = false
	conf.Secret = ""
	conf.JpegDefaults.Quality = 90
	conf.WebpDefaults.Quality = 90
	conf.WebpDefaults.Lossless = false

	_, err := toml.DecodeFile(filename, &conf)

	if conf.Secret == "" {
		log.Fatalln("Config: \"secret\" missing")
	}

	return conf, err
}
