package caching

import (
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestS3Config_concatFilename(t *testing.T) {
	backend := S3Config{
		PathPrefix: "cache-prefix",
	}

	assert.Equal(t, filepath.Join("cache-prefix", "originals", "test.jpg"), backend.concatFilename("/originals/test.jpg"))
}

func TestS3Config_NewSessionDoesNotRequireNetwork(t *testing.T) {
	backend := S3Config{
		Endpoint: "http://127.0.0.1:9000",
		Region:   "test-region",
		Key:      "test-key",
		Secret:   "test-secret",
	}

	session, err := backend.NewSession()
	assert.Nil(t, err)
	assert.NotNil(t, session)
}
