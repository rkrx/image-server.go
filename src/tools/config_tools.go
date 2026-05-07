package tools

import (
	"crypto/md5"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type autocrop struct {
	Apply     bool
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
	Name             string
	OriginalBackend  string
	ImageOriginalDir string
	ImageCacheDir    string
	Type             FileType
	Width            int
	Height           int
	Mode             string
	Sharpen          float32
	Autocrop         autocrop
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

func (profile *Profile) ApplyFromProfile(otherProfile Profile) {
	if otherProfile.Name != "" {
		profile.Name = otherProfile.Name
	}

	if otherProfile.OriginalBackend != "" {
		profile.OriginalBackend = otherProfile.OriginalBackend
	}

	if otherProfile.ImageOriginalDir != "" {
		profile.ImageOriginalDir = otherProfile.ImageOriginalDir
	}

	if otherProfile.ImageCacheDir != "" {
		profile.ImageCacheDir = otherProfile.ImageCacheDir
	}

	if otherProfile.Type != "" { // Assuming FileType is a string-like type
		profile.Type = otherProfile.Type
	}

	if otherProfile.Width != 0 {
		profile.Width = otherProfile.Width
	}

	if otherProfile.Height != 0 {
		profile.Height = otherProfile.Height
	}

	if otherProfile.Mode != "" {
		profile.Mode = otherProfile.Mode
	}

	if otherProfile.Sharpen != 0 {
		profile.Sharpen = otherProfile.Sharpen
	}

	// Assuming `autocrop` is a struct or another type that should be copied as is
	if otherProfile.Autocrop != (autocrop{}) {
		profile.Autocrop = otherProfile.Autocrop
	}
}

func (profile Profile) GetDirectoryHash() string {
	var name = "dynamic"
	if profile.Name != "" {
		name = fmt.Sprintf("name=%s", profile.Name)
	}
	var backendHash = fmt.Sprintf("%x", md5.Sum([]byte(profile.OriginalBackend)))
	return fmt.Sprintf("backend-hash=%s;%s;%d-%d;m=%s;sh=%.3f;ac=%.3f,%.3f",
		backendHash,
		name,
		profile.Width,
		profile.Height,
		profile.Mode,
		profile.Sharpen,
		profile.Autocrop.Tolerance,
		profile.Autocrop.Border,
	)
}

type ServerType struct {
	ListenAddr             string
	OriginalBackend        string
	ImageProcessingWorkers int
	RemoteAddrHeader       string
}

type JpegDefaults struct {
	Quality int `toml:"quality"`
}

type WebpDefaults struct {
	Quality  int  `toml:"quality"`
	Lossless bool `toml:"lossless"`
}

type Config struct {
	LoggingEnabled                     bool         `toml:"enableLogging"`
	Secret                             string       `toml:"secret"`
	HashAlgorithm                      string       `toml:"hashAlgorithm"`
	HashLength                         int          `toml:"hashLength"`
	OriginFetchTimeoutInSeconds        int          `toml:"originFetchTimeoutInSeconds"`
	OriginMaxImageBytes                int64        `toml:"originMaxImageBytes"`
	DisableHashChecking                bool         `toml:"disableHashChecking"`
	DisableHashCheckingFromIPv4Sources []string     `toml:"disableHashCheckingFromIPv4Sources"`
	DisableCaching                     bool         `toml:"disableCaching"`
	HashCacheFilenames                 bool         `toml:"hashCacheFilenames"`
	ImageOriginalDir                   string       `toml:"imageOriginalDir"`
	ImageCacheDir                      string       `toml:"imageCacheDir"`
	CacheMaxAgeInSeconds               int32        `toml:"cacheMaxAgeInSeconds"`
	Server                             ServerType   `toml:"server"`
	Profiles                           []Profile    `toml:"profile"`
	JpegDefaults                       JpegDefaults `toml:"jpegDefaults"`
	WebpDefaults                       WebpDefaults `toml:"webpDefaults"`
}

func (conf Config) GetProfile(name string) (Profile, error) {
	for _, profile := range conf.Profiles {
		if profile.Name == name {
			profile = conf.SetProfileDefaults(profile)
			return profile, nil
		}
	}
	return conf.GetDefaultProfile(), errors.New("profile not found")
}

func (conf Config) SetProfileDefaults(profile Profile) Profile {
	if profile.Type == "" {
		profile.Type = Jpeg
	}
	if profile.Mode == "" {
		profile.Mode = "contain"
	}
	if profile.OriginalBackend == "" {
		profile.OriginalBackend = conf.Server.OriginalBackend
	}
	if profile.ImageOriginalDir == "" {
		profile.ImageOriginalDir = conf.ImageOriginalDir
	}
	if profile.ImageCacheDir == "" {
		profile.ImageCacheDir = conf.ImageCacheDir
	}
	return profile
}

func (conf Config) GetDefaultProfile() Profile {
	return conf.SetProfileDefaults(Profile{})
}

func LoadConfig(filename string) (Config, error) {
	var conf Config

	coalesce := func(str ...string) string {
		for _, s := range str {
			if s != "" {
				return s
			}
		}
		return ""
	}

	conf.LoggingEnabled = true
	conf.HashCacheFilenames = false
	conf.Secret = ""
	conf.HashAlgorithm = "md5"
	conf.HashLength = 8
	conf.OriginFetchTimeoutInSeconds = 15
	conf.OriginMaxImageBytes = 50 << 20

	conf.ImageOriginalDir = coalesce(
		os.Getenv("GO_IMG_SRV_VAR_ORIGINALS_DIR"),
		filepath.Join(os.Getenv("GO_IMG_SRV_VAR_CACHE_DIR"), "originals"),
		"var/cache/originals",
	)

	if !FileExists(conf.ImageOriginalDir) {
		if err := os.MkdirAll(conf.ImageOriginalDir, 0777); err != nil {
			log.Fatal(err)
		}
	}

	conf.ImageCacheDir = coalesce(
		conf.ImageCacheDir,
		os.Getenv("GO_IMG_SRV_VAR_IMAGES_DIR"),
		filepath.Join(os.Getenv("GO_IMG_SRV_VAR_CACHE_DIR"), "images"),
		"var/cache/images",
	)

	if !FileExists(conf.ImageCacheDir) {
		if err := os.MkdirAll(conf.ImageCacheDir, 0777); err != nil {
			log.Fatal(err)
		}
	}

	conf.JpegDefaults.Quality = 90
	conf.WebpDefaults.Quality = 90
	conf.WebpDefaults.Lossless = false

	_, err := toml.DecodeFile(filename, &conf)

	if conf.Secret == "" {
		log.Fatalln("Config: \"secret\" missing")
	}

	return conf, err
}
