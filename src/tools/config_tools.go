package tools

import (
	"crypto/md5"
	"errors"
	"fmt"
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
	Avif FileType = "avif"
	Png  FileType = "png"
	Gif  FileType = "gif"
)

const (
	ImageCacheSchemaVersion = 2
	DefaultJpegQuality      = 85
	DefaultWebpQuality      = 80
	DefaultAvifQuality      = 60
	DefaultAvifAlphaQuality = 80
	DefaultAvifSpeed        = 6

	DefaultImageProcessingWorkers           = 2
	DefaultMaxSourceImageWidth              = 12000
	DefaultMaxSourceImageHeight             = 12000
	DefaultMaxSourceImagePixels       int64 = 40000000
	DefaultMaxOutputImageWidth              = 8192
	DefaultMaxOutputImageHeight             = 8192
	DefaultMaxOutputImagePixels       int64 = 20000000
	DefaultMaxCompositionNodes              = 64
	DefaultMaxCompositionImages             = 32
	DefaultMaxCompositionDepth              = 8
	DefaultMaxCompositionSourcePixels int64 = 80000000
	DefaultMaxCompositionRenderPixels int64 = 80000000
	DefaultMaxRequestURIBytes               = 8192
	DefaultMaxQueryParameters               = 16
	DefaultReadHeaderTimeoutInSeconds       = 5
	DefaultReadTimeoutInSeconds             = 15
	DefaultWriteTimeoutInSeconds            = 120
	DefaultIdleTimeoutInSeconds             = 60
	DefaultMaxHeaderBytes                   = 64 << 10
	DefaultMaxConcurrentStreams             = 100
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

func (fileType FileType) ContentType() string {
	switch fileType {
	case Jpeg:
		return "image/jpeg"
	case WebP:
		return "image/webp"
	case Avif:
		return "image/avif"
	case Png:
		return "image/png"
	case Gif:
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}

// ApplyFileType Sets Profile.Type to a valid file type based on string filetype. If fileType is not recognized or
// empty, the currently set file type will remain untoucher or - if empty - set to Jpeg.
func (profile *Profile) ApplyFileType(fileType string) error {
	switch strings.ToLower(fileType) {
	case "", "jpg", "jpeg":
		profile.Type = Jpeg
	case "webp":
		profile.Type = WebP
	case "avif":
		profile.Type = Avif
	case "png":
		profile.Type = Png
	case "gif":
		profile.Type = Gif
	default:
		if profile.Type == "" {
			profile.Type = Jpeg
		}
		return fmt.Errorf("unknown file type: %s", fileType)
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

func (profile Profile) GetDirectoryHash(config Config) string {
	var name = "dynamic"
	if profile.Name != "" {
		name = fmt.Sprintf("name=%s", profile.Name)
	}
	var backendHash = fmt.Sprintf("%x", md5.Sum([]byte(profile.OriginalBackend)))
	return fmt.Sprintf("cache-v=%d;backend-hash=%s;%s;%d-%d;m=%s;sh=%.3f;ac=%.3f,%.3f;t=%s;jq=%d;wq=%d;wl=%t;aq=%d;aaq=%d;as=%d",
		ImageCacheSchemaVersion,
		backendHash,
		name,
		profile.Width,
		profile.Height,
		profile.Mode,
		profile.Sharpen,
		profile.Autocrop.Tolerance,
		profile.Autocrop.Border,
		profile.Type,
		config.JpegDefaults.Quality,
		config.WebpDefaults.Quality,
		config.WebpDefaults.Lossless,
		config.AvifDefaults.Quality,
		config.AvifDefaults.AlphaQuality,
		config.AvifDefaults.Speed,
	)
}

type ServerType struct {
	ListenAddr                 string `toml:"listenAddr"`
	OriginalBackend            string `toml:"originalBackend"`
	ImageProcessingWorkers     int    `toml:"imageProcessingWorkers"`
	RemoteAddrHeader           string `toml:"remoteAddrHeader"`
	ReadHeaderTimeoutInSeconds int    `toml:"readHeaderTimeoutInSeconds"`
	ReadTimeoutInSeconds       int    `toml:"readTimeoutInSeconds"`
	WriteTimeoutInSeconds      int    `toml:"writeTimeoutInSeconds"`
	IdleTimeoutInSeconds       int    `toml:"idleTimeoutInSeconds"`
	MaxHeaderBytes             int    `toml:"maxHeaderBytes"`
	MaxRequestURIBytes         int    `toml:"maxRequestURIBytes"`
	MaxQueryParameters         int    `toml:"maxQueryParameters"`
	MaxConcurrentHTTP2Streams  int    `toml:"maxConcurrentHTTP2Streams"`
}

type JpegDefaults struct {
	Quality int `toml:"quality"`
}

type WebpDefaults struct {
	Quality  int  `toml:"quality"`
	Lossless bool `toml:"lossless"`
}

type AvifDefaults struct {
	Quality      int `toml:"quality"`
	AlphaQuality int `toml:"alphaQuality"`
	Speed        int `toml:"speed"`
}

type Config struct {
	LoggingEnabled                     bool         `toml:"enableLogging"`
	Secret                             string       `toml:"secret"`
	HashAlgorithm                      string       `toml:"hashAlgorithm"`
	HashLength                         int          `toml:"hashLength"`
	OriginFetchTimeoutInSeconds        int          `toml:"originFetchTimeoutInSeconds"`
	OriginMaxImageBytes                int64        `toml:"originMaxImageBytes"`
	MaxSourceImageWidth                int          `toml:"maxSourceImageWidth"`
	MaxSourceImageHeight               int          `toml:"maxSourceImageHeight"`
	MaxSourceImagePixels               int64        `toml:"maxSourceImagePixels"`
	MaxOutputImageWidth                int          `toml:"maxOutputImageWidth"`
	MaxOutputImageHeight               int          `toml:"maxOutputImageHeight"`
	MaxOutputImagePixels               int64        `toml:"maxOutputImagePixels"`
	MaxCompositionNodes                int          `toml:"maxCompositionNodes"`
	MaxCompositionImages               int          `toml:"maxCompositionImages"`
	MaxCompositionDepth                int          `toml:"maxCompositionDepth"`
	MaxCompositionSourcePixels         int64        `toml:"maxCompositionSourcePixels"`
	MaxCompositionRenderPixels         int64        `toml:"maxCompositionRenderPixels"`
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
	AvifDefaults                       AvifDefaults `toml:"avifDefaults"`
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

func (conf Config) ValidateSourceImageDimensions(width, height int) error {
	return validateImageDimensions(
		width,
		height,
		positiveOrDefault(conf.MaxSourceImageWidth, DefaultMaxSourceImageWidth),
		positiveOrDefault(conf.MaxSourceImageHeight, DefaultMaxSourceImageHeight),
		positiveInt64OrDefault(conf.MaxSourceImagePixels, DefaultMaxSourceImagePixels),
	)
}

func (conf Config) ValidateOutputImageDimensions(width, height int) error {
	return validateImageDimensions(
		width,
		height,
		positiveOrDefault(conf.MaxOutputImageWidth, DefaultMaxOutputImageWidth),
		positiveOrDefault(conf.MaxOutputImageHeight, DefaultMaxOutputImageHeight),
		positiveInt64OrDefault(conf.MaxOutputImagePixels, DefaultMaxOutputImagePixels),
	)
}

func (conf Config) ValidateRequestedOutputDimensions(width, height int) error {
	if width < 0 || height < 0 {
		return errors.New("image dimensions must not be negative")
	}
	if width > 0 && width > positiveOrDefault(conf.MaxOutputImageWidth, DefaultMaxOutputImageWidth) {
		return fmt.Errorf("requested image width %d exceeds the configured limit", width)
	}
	if height > 0 && height > positiveOrDefault(conf.MaxOutputImageHeight, DefaultMaxOutputImageHeight) {
		return fmt.Errorf("requested image height %d exceeds the configured limit", height)
	}
	if width > 0 && height > 0 && exceedsPixelLimit(width, height, positiveInt64OrDefault(conf.MaxOutputImagePixels, DefaultMaxOutputImagePixels)) {
		return fmt.Errorf("requested image dimensions %dx%d exceed the configured pixel limit", width, height)
	}
	return nil
}

func validateImageDimensions(width, height, maxWidth, maxHeight int, maxPixels int64) error {
	if width < 1 || height < 1 {
		return fmt.Errorf("invalid image dimensions %dx%d", width, height)
	}
	if width > maxWidth || height > maxHeight {
		return fmt.Errorf("image dimensions %dx%d exceed the configured limit %dx%d", width, height, maxWidth, maxHeight)
	}
	if exceedsPixelLimit(width, height, maxPixels) {
		return fmt.Errorf("image dimensions %dx%d exceed the configured pixel limit", width, height)
	}
	return nil
}

func exceedsPixelLimit(width, height int, maxPixels int64) bool {
	return int64(width) > maxPixels/int64(height)
}

func positiveOrDefault(value, defaultValue int) int {
	if value > 0 {
		return value
	}
	return defaultValue
}

func positiveInt64OrDefault(value, defaultValue int64) int64 {
	if value > 0 {
		return value
	}
	return defaultValue
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
	conf.MaxSourceImageWidth = DefaultMaxSourceImageWidth
	conf.MaxSourceImageHeight = DefaultMaxSourceImageHeight
	conf.MaxSourceImagePixels = DefaultMaxSourceImagePixels
	conf.MaxOutputImageWidth = DefaultMaxOutputImageWidth
	conf.MaxOutputImageHeight = DefaultMaxOutputImageHeight
	conf.MaxOutputImagePixels = DefaultMaxOutputImagePixels
	conf.MaxCompositionNodes = DefaultMaxCompositionNodes
	conf.MaxCompositionImages = DefaultMaxCompositionImages
	conf.MaxCompositionDepth = DefaultMaxCompositionDepth
	conf.MaxCompositionSourcePixels = DefaultMaxCompositionSourcePixels
	conf.MaxCompositionRenderPixels = DefaultMaxCompositionRenderPixels
	conf.Server.ImageProcessingWorkers = DefaultImageProcessingWorkers
	conf.Server.ReadHeaderTimeoutInSeconds = DefaultReadHeaderTimeoutInSeconds
	conf.Server.ReadTimeoutInSeconds = DefaultReadTimeoutInSeconds
	conf.Server.WriteTimeoutInSeconds = DefaultWriteTimeoutInSeconds
	conf.Server.IdleTimeoutInSeconds = DefaultIdleTimeoutInSeconds
	conf.Server.MaxHeaderBytes = DefaultMaxHeaderBytes
	conf.Server.MaxRequestURIBytes = DefaultMaxRequestURIBytes
	conf.Server.MaxQueryParameters = DefaultMaxQueryParameters
	conf.Server.MaxConcurrentHTTP2Streams = DefaultMaxConcurrentStreams

	conf.ImageOriginalDir = coalesce(
		os.Getenv("GO_IMG_SRV_VAR_ORIGINALS_DIR"),
		filepath.Join(os.Getenv("GO_IMG_SRV_VAR_CACHE_DIR"), "originals"),
		"var/cache/originals",
	)

	conf.ImageCacheDir = coalesce(
		conf.ImageCacheDir,
		os.Getenv("GO_IMG_SRV_VAR_IMAGES_DIR"),
		filepath.Join(os.Getenv("GO_IMG_SRV_VAR_CACHE_DIR"), "images"),
		"var/cache/images",
	)

	conf.JpegDefaults.Quality = DefaultJpegQuality
	conf.WebpDefaults.Quality = DefaultWebpQuality
	conf.WebpDefaults.Lossless = false
	conf.AvifDefaults.Quality = DefaultAvifQuality
	conf.AvifDefaults.AlphaQuality = DefaultAvifAlphaQuality
	conf.AvifDefaults.Speed = DefaultAvifSpeed

	if _, err := toml.DecodeFile(filename, &conf); err != nil {
		return conf, err
	}

	if secret := os.Getenv("GO_IMG_SRV_SECRET"); secret != "" {
		conf.Secret = secret
	}

	if conf.Secret == "" {
		return conf, errors.New("configuration secret missing: set GO_IMG_SRV_SECRET or secret in config.toml")
	}
	applySafeDefaults(&conf)
	if err := validateConfig(conf); err != nil {
		return conf, err
	}

	if err := ensurePrivateDirectory(conf.ImageOriginalDir); err != nil {
		return conf, fmt.Errorf("prepare original image directory: %w", err)
	}
	if err := ensurePrivateDirectory(conf.ImageCacheDir); err != nil {
		return conf, fmt.Errorf("prepare image cache directory: %w", err)
	}

	return conf, nil
}

func ensurePrivateDirectory(path string) error {
	if err := os.MkdirAll(path, 0750); err != nil {
		return err
	}
	// #nosec G302 -- cache directories require execute permission for traversal.
	return os.Chmod(path, 0750)
}

func applySafeDefaults(conf *Config) {
	conf.OriginFetchTimeoutInSeconds = positiveOrDefault(conf.OriginFetchTimeoutInSeconds, 15)
	conf.OriginMaxImageBytes = positiveInt64OrDefault(conf.OriginMaxImageBytes, 50<<20)
	conf.MaxSourceImageWidth = positiveOrDefault(conf.MaxSourceImageWidth, DefaultMaxSourceImageWidth)
	conf.MaxSourceImageHeight = positiveOrDefault(conf.MaxSourceImageHeight, DefaultMaxSourceImageHeight)
	conf.MaxSourceImagePixels = positiveInt64OrDefault(conf.MaxSourceImagePixels, DefaultMaxSourceImagePixels)
	conf.MaxOutputImageWidth = positiveOrDefault(conf.MaxOutputImageWidth, DefaultMaxOutputImageWidth)
	conf.MaxOutputImageHeight = positiveOrDefault(conf.MaxOutputImageHeight, DefaultMaxOutputImageHeight)
	conf.MaxOutputImagePixels = positiveInt64OrDefault(conf.MaxOutputImagePixels, DefaultMaxOutputImagePixels)
	conf.MaxCompositionNodes = positiveOrDefault(conf.MaxCompositionNodes, DefaultMaxCompositionNodes)
	conf.MaxCompositionImages = positiveOrDefault(conf.MaxCompositionImages, DefaultMaxCompositionImages)
	conf.MaxCompositionDepth = positiveOrDefault(conf.MaxCompositionDepth, DefaultMaxCompositionDepth)
	conf.MaxCompositionSourcePixels = positiveInt64OrDefault(conf.MaxCompositionSourcePixels, DefaultMaxCompositionSourcePixels)
	conf.MaxCompositionRenderPixels = positiveInt64OrDefault(conf.MaxCompositionRenderPixels, DefaultMaxCompositionRenderPixels)
	conf.Server.ImageProcessingWorkers = positiveOrDefault(conf.Server.ImageProcessingWorkers, DefaultImageProcessingWorkers)
	conf.Server.ReadHeaderTimeoutInSeconds = positiveOrDefault(conf.Server.ReadHeaderTimeoutInSeconds, DefaultReadHeaderTimeoutInSeconds)
	conf.Server.ReadTimeoutInSeconds = positiveOrDefault(conf.Server.ReadTimeoutInSeconds, DefaultReadTimeoutInSeconds)
	conf.Server.WriteTimeoutInSeconds = positiveOrDefault(conf.Server.WriteTimeoutInSeconds, DefaultWriteTimeoutInSeconds)
	conf.Server.IdleTimeoutInSeconds = positiveOrDefault(conf.Server.IdleTimeoutInSeconds, DefaultIdleTimeoutInSeconds)
	conf.Server.MaxHeaderBytes = positiveOrDefault(conf.Server.MaxHeaderBytes, DefaultMaxHeaderBytes)
	conf.Server.MaxRequestURIBytes = positiveOrDefault(conf.Server.MaxRequestURIBytes, DefaultMaxRequestURIBytes)
	conf.Server.MaxQueryParameters = positiveOrDefault(conf.Server.MaxQueryParameters, DefaultMaxQueryParameters)
	conf.Server.MaxConcurrentHTTP2Streams = positiveOrDefault(conf.Server.MaxConcurrentHTTP2Streams, DefaultMaxConcurrentStreams)
}

func validateConfig(conf Config) error {
	algorithm := strings.ToLower(conf.HashAlgorithm)
	if algorithm != "md5" && algorithm != "sha256" && algorithm != "hmac-sha256" {
		return fmt.Errorf("unsupported hash algorithm %q", conf.HashAlgorithm)
	}
	maxHashLength := 64
	if algorithm == "md5" {
		maxHashLength = 32
	}
	if conf.HashLength < 8 || conf.HashLength > maxHashLength {
		return fmt.Errorf("hashLength must be between 8 and %d for %s", maxHashLength, algorithm)
	}
	if conf.Server.ImageProcessingWorkers > 64 {
		return errors.New("imageProcessingWorkers must not exceed 64")
	}
	if conf.Server.MaxConcurrentHTTP2Streams > 10000 {
		return errors.New("maxConcurrentHTTP2Streams must not exceed 10000")
	}
	if conf.JpegDefaults.Quality < 1 || conf.JpegDefaults.Quality > 100 {
		return errors.New("jpeg quality must be between 1 and 100")
	}
	if conf.WebpDefaults.Quality < 0 || conf.WebpDefaults.Quality > 100 {
		return errors.New("webp quality must be between 0 and 100")
	}
	if conf.AvifDefaults.Quality < 0 || conf.AvifDefaults.Quality > 100 || conf.AvifDefaults.AlphaQuality < 0 || conf.AvifDefaults.AlphaQuality > 100 {
		return errors.New("AVIF quality values must be between 0 and 100")
	}
	if conf.AvifDefaults.Speed < 0 || conf.AvifDefaults.Speed > 10 {
		return errors.New("AVIF speed must be between 0 and 10")
	}
	for _, profile := range conf.Profiles {
		if err := conf.ValidateRequestedOutputDimensions(profile.Width, profile.Height); err != nil {
			return fmt.Errorf("profile %q: %w", profile.Name, err)
		}
		if profile.Sharpen < 0 || profile.Sharpen > 10 {
			return fmt.Errorf("profile %q: sharpen must be between 0 and 10", profile.Name)
		}
		if profile.Autocrop.Tolerance < 0 || profile.Autocrop.Tolerance > 100 || profile.Autocrop.Border < 0 || profile.Autocrop.Border > 25 {
			return fmt.Errorf("profile %q: invalid autocrop limits", profile.Name)
		}
	}
	return nil
}
