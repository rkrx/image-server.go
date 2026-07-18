package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	conf, cacheDir := loadConfigForTest(t)
	assert.True(t, conf.LoggingEnabled)
	assert.True(t, conf.HashCacheFilenames)
	assert.Equal(t, "md5", conf.HashAlgorithm)
	assert.Equal(t, 8, conf.HashLength)
	assert.Equal(t, 15, conf.OriginFetchTimeoutInSeconds)
	assert.Equal(t, int64(50<<20), conf.OriginMaxImageBytes)
	assert.Equal(t, DefaultMaxSourceImageWidth, conf.MaxSourceImageWidth)
	assert.Equal(t, DefaultMaxSourceImageHeight, conf.MaxSourceImageHeight)
	assert.Equal(t, DefaultMaxSourceImagePixels, conf.MaxSourceImagePixels)
	assert.Equal(t, DefaultMaxOutputImageWidth, conf.MaxOutputImageWidth)
	assert.Equal(t, DefaultMaxOutputImageHeight, conf.MaxOutputImageHeight)
	assert.Equal(t, DefaultMaxOutputImagePixels, conf.MaxOutputImagePixels)
	assert.Equal(t, filepath.Join(cacheDir, "originals"), conf.ImageOriginalDir)
	assert.Equal(t, filepath.Join(cacheDir, "images"), conf.ImageCacheDir)
	assert.True(t, FileExists(conf.ImageOriginalDir))
	assert.True(t, FileExists(conf.ImageCacheDir))
	originalDirInfo, err := os.Stat(conf.ImageOriginalDir)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0750), originalDirInfo.Mode().Perm())
	cacheDirInfo, err := os.Stat(conf.ImageCacheDir)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0750), cacheDirInfo.Mode().Perm())

	assert.Equal(t, 85, conf.JpegDefaults.Quality)
	assert.Equal(t, 80, conf.WebpDefaults.Quality)
	assert.True(t, conf.WebpDefaults.Lossless)
	assert.Equal(t, 55, conf.AvifDefaults.Quality)
	assert.Equal(t, 75, conf.AvifDefaults.AlphaQuality)
	assert.Equal(t, 8, conf.AvifDefaults.Speed)

	assert.Equal(t, "0.0.0.0:8080", conf.Server.ListenAddr)
	assert.Equal(t, "https://127.0.0.1/original", conf.Server.OriginalBackend)
	assert.Equal(t, DefaultImageProcessingWorkers, conf.Server.ImageProcessingWorkers)
	assert.Equal(t, DefaultReadHeaderTimeoutInSeconds, conf.Server.ReadHeaderTimeoutInSeconds)
	assert.Equal(t, DefaultReadTimeoutInSeconds, conf.Server.ReadTimeoutInSeconds)
	assert.Equal(t, DefaultWriteTimeoutInSeconds, conf.Server.WriteTimeoutInSeconds)
	assert.Equal(t, DefaultIdleTimeoutInSeconds, conf.Server.IdleTimeoutInSeconds)
	assert.Equal(t, DefaultMaxHeaderBytes, conf.Server.MaxHeaderBytes)
	assert.Equal(t, DefaultMaxRequestURIBytes, conf.Server.MaxRequestURIBytes)
	assert.Equal(t, DefaultMaxQueryParameters, conf.Server.MaxQueryParameters)
	assert.Equal(t, DefaultMaxConcurrentStreams, conf.Server.MaxConcurrentHTTP2Streams)

	assert.Equal(t, Profile{Name: "500px", Width: 500, Height: 500}, conf.Profiles[0])
	assert.Equal(t, Profile{Name: "250px", Width: 250, Height: 250, Sharpen: 0.6, Autocrop: autocrop{Tolerance: 15, Border: 3}}, conf.Profiles[1])
}

func TestConfig_GetProfile(t *testing.T) {
	conf, cacheDir := loadConfigForTest(t)
	profile, err := conf.GetProfile("250px")
	assert.Nil(t, err)

	expectedProfile := Profile{
		Name:             "250px",
		OriginalBackend:  "https://127.0.0.1/original",
		ImageOriginalDir: filepath.Join(cacheDir, "originals"),
		ImageCacheDir:    filepath.Join(cacheDir, "images"),
		Type:             Jpeg,
		Width:            250,
		Height:           250,
		Mode:             "contain",
		Sharpen:          0.6,
		Autocrop:         autocrop{Tolerance: 15, Border: 3},
	}

	assert.Equal(t, expectedProfile, profile)
}

func loadConfigForTest(t *testing.T) (Config, string) {
	t.Helper()
	cacheDir := t.TempDir()
	t.Setenv("GO_IMG_SRV_VAR_CACHE_DIR", cacheDir)
	t.Setenv("GO_IMG_SRV_VAR_ORIGINALS_DIR", "")
	t.Setenv("GO_IMG_SRV_VAR_IMAGES_DIR", "")

	conf, err := LoadConfig("test_assets/config.toml")
	assert.Nil(t, err)
	return conf, cacheDir
}

func TestProfile_GetDirectoryHash(t *testing.T) {
	profile := Profile{
		Name:            "test",
		OriginalBackend: "https://127.0.0.1/original",
		Type:            Jpeg,
		Width:           250,
		Height:          250,
		Mode:            "contain",
		Sharpen:         .6,
		Autocrop:        autocrop{Tolerance: 15, Border: 3},
	}
	config := Config{
		JpegDefaults: JpegDefaults{Quality: 85},
		WebpDefaults: WebpDefaults{Quality: 80, Lossless: false},
		AvifDefaults: AvifDefaults{Quality: 60, AlphaQuality: 80, Speed: 6},
	}
	assert.Equal(t, "cache-v=2;backend-hash=a73b283ad646cf547c4cbbaa334a342d;name=test;250-250;m=contain;sh=0.600;ac=15.000,3.000;t=jpeg;jq=85;wq=80;wl=false;aq=60;aaq=80;as=6", profile.GetDirectoryHash(config))

	initialCacheKey := profile.GetDirectoryHash(config)
	config.WebpDefaults.Quality++
	assert.NotEqual(t, initialCacheKey, profile.GetDirectoryHash(config))
	config.WebpDefaults.Quality--
	config.AvifDefaults.Quality++
	assert.NotEqual(t, initialCacheKey, profile.GetDirectoryHash(config))
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
	err = profile.ApplyFileType("avif")
	assert.Nil(t, err)
	assert.Equal(t, Avif, profile.Type)

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

func TestLoadConfigUsesEncoderDefaults(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("GO_IMG_SRV_VAR_CACHE_DIR", cacheDir)
	t.Setenv("GO_IMG_SRV_VAR_ORIGINALS_DIR", "")
	t.Setenv("GO_IMG_SRV_VAR_IMAGES_DIR", "")

	configPath := filepath.Join(t.TempDir(), "config.toml")
	assert.NoError(t, os.WriteFile(configPath, []byte("secret = \"test\"\n"), 0600))

	config, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.Equal(t, DefaultJpegQuality, config.JpegDefaults.Quality)
	assert.Equal(t, DefaultWebpQuality, config.WebpDefaults.Quality)
	assert.False(t, config.WebpDefaults.Lossless)
	assert.Equal(t, DefaultAvifQuality, config.AvifDefaults.Quality)
	assert.Equal(t, DefaultAvifAlphaQuality, config.AvifDefaults.AlphaQuality)
	assert.Equal(t, DefaultAvifSpeed, config.AvifDefaults.Speed)
}

func TestLoadConfigUsesEnvironmentSecret(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("GO_IMG_SRV_VAR_CACHE_DIR", cacheDir)
	t.Setenv("GO_IMG_SRV_SECRET", "environment-secret")

	configPath := filepath.Join(t.TempDir(), "config.toml")
	assert.NoError(t, os.WriteFile(configPath, []byte("enableLogging = false\n"), 0600))

	config, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.Equal(t, "environment-secret", config.Secret)
}

func TestLoadConfigEnvironmentSecretOverridesFile(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("GO_IMG_SRV_VAR_CACHE_DIR", cacheDir)
	t.Setenv("GO_IMG_SRV_SECRET", "environment-secret")

	configPath := filepath.Join(t.TempDir(), "config.toml")
	assert.NoError(t, os.WriteFile(configPath, []byte("secret = \"file-secret\"\n"), 0600))

	config, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.Equal(t, "environment-secret", config.Secret)
}

func TestLoadConfigRequiresSecret(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("GO_IMG_SRV_VAR_CACHE_DIR", cacheDir)
	t.Setenv("GO_IMG_SRV_SECRET", "")

	configPath := filepath.Join(t.TempDir(), "config.toml")
	assert.NoError(t, os.WriteFile(configPath, []byte("enableLogging = false\n"), 0600))

	_, err := LoadConfig(configPath)
	assert.ErrorContains(t, err, "GO_IMG_SRV_SECRET")
}

func TestSampleConfig(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("GO_IMG_SRV_VAR_CACHE_DIR", cacheDir)
	t.Setenv("GO_IMG_SRV_VAR_ORIGINALS_DIR", "")
	t.Setenv("GO_IMG_SRV_VAR_IMAGES_DIR", "")
	t.Setenv("GO_IMG_SRV_SECRET", "sample-config-test-secret")

	config, err := LoadConfig("../../config.sample.toml")
	assert.NoError(t, err)
	assert.Equal(t, "sha256", config.HashAlgorithm)
	assert.Equal(t, DefaultJpegQuality, config.JpegDefaults.Quality)
	assert.Equal(t, DefaultWebpQuality, config.WebpDefaults.Quality)
	assert.Equal(t, DefaultAvifQuality, config.AvifDefaults.Quality)
	assert.Equal(t, DefaultMaxSourceImagePixels, config.MaxSourceImagePixels)
	assert.Equal(t, DefaultMaxOutputImagePixels, config.MaxOutputImagePixels)
	assert.Equal(t, DefaultImageProcessingWorkers, config.Server.ImageProcessingWorkers)

	productCard, err := config.GetProfile("product-card")
	assert.NoError(t, err)
	assert.Equal(t, 480, productCard.Width)
	assert.Equal(t, 480, productCard.Height)
	assert.Equal(t, "cover", productCard.Mode)
	assert.True(t, productCard.Autocrop.Apply)
}
