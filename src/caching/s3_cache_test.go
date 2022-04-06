package caching

import (
	"bufio"
	"bytes"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"testing"
)

func TestS3Backend_Has(t *testing.T) {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	backend := S3Config{
		Endpoint:   os.Getenv("S3_ENDPOINT"),
		Bucket:     os.Getenv("S3_BUCKET"),
		PathPrefix: os.Getenv("S3_PATH_PREFIX"),
		Region:     os.Getenv("S3_REGION"),
		Key:        os.Getenv("S3_KEY"),
		Secret:     os.Getenv("S3_SECRET"),
	}

	result, err := backend.Has("/originals/0b1cb2f5f0ead264aec021970e3f9cbe.jpg")
	assert.Nil(t, err)
	assert.True(t, result)

	result, err = backend.Has("/originals/abc123.jpg")
	assert.Nil(t, err)
	assert.False(t, result)
}

func TestS3Backend_Get(t *testing.T) {
	var err error
	err = godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	backend := S3Config{
		Endpoint:   os.Getenv("S3_ENDPOINT"),
		Bucket:     os.Getenv("S3_BUCKET"),
		PathPrefix: os.Getenv("S3_PATH_PREFIX"),
		Region:     os.Getenv("S3_REGION"),
		Key:        os.Getenv("S3_KEY"),
		Secret:     os.Getenv("S3_SECRET"),
		PublicAddr: os.Getenv("S3_PUBLIC_ADDR"),
	}

	buf := bytes.Buffer{}
	writer := bufio.NewWriter(&buf)
	_, err = backend.Get("/originals/0b1cb2f5f0ead264aec021970e3f9cbe.jpg", writer)
	assert.Nil(t, err)
	assert.Equal(t, 18731, buf.Len())
}

func TestS3Backend_Put(t *testing.T) {
	var err error
	err = godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	backend := S3Config{
		Endpoint:   os.Getenv("S3_ENDPOINT"),
		Bucket:     os.Getenv("S3_BUCKET"),
		PathPrefix: os.Getenv("S3_PATH_PREFIX"),
		Region:     os.Getenv("S3_REGION"),
		Key:        os.Getenv("S3_KEY"),
		Secret:     os.Getenv("S3_SECRET"),
		PublicAddr: os.Getenv("S3_PUBLIC_ADDR"),
	}

	b, err := os.ReadFile("../imaging/test_assets/test32.jpeg")
	assert.Nil(t, err)
	reader := bytes.NewReader(b)
	err = backend.Put("/originals/test.jpg", "image/jpeg", reader)
	assert.Nil(t, err)
}
