package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image-server/src/tools"
	"net/url"
	"strings"
)

func extractHashFromRequestUri(requestUri string, values url.Values, config tools.Config) bool {
	if values.Has("hash") {
		var hash, expectedHash string
		success, groups := tools.RegexMatch(requestUri, "\\?.*?[&\\?]hash=([a-f0-9]+)(&|$)")
		if success {
			hash = groups[1]

			requestUriWoHash := tools.RegexReplace(requestUri, "(\\?.*?)([&\\?]hash=(?:[a-f0-9]+))(&|$)", "$1$3")
			expectedHash = generateRequestHash(requestUriWoHash, config)
			if compareRequestHash(expectedHash, hash) {
				return true
			}

			requestUriWoHashAndType := tools.RegexReplace(requestUriWoHash, "(\\?.*?)([&\\?]type=[a-z]+)(&|$)", "$1$3")
			expectedHash = generateRequestHash(requestUriWoHashAndType, config)
			if compareRequestHash(expectedHash, hash) {
				return true
			}
		}
	}
	return false
}

func compareRequestHash(expectedHash, hash string) bool {
	return hmac.Equal([]byte(expectedHash), []byte(hash))
}

func generateRequestHash(requestUri string, config tools.Config) string {
	algorithm := strings.ToLower(config.HashAlgorithm)
	if algorithm == "" {
		algorithm = "md5"
	}

	hashLength := config.HashLength
	if hashLength < 1 {
		hashLength = 8
	}

	var hash string
	switch algorithm {
	case "sha256", "hmac-sha256":
		mac := hmac.New(sha256.New, []byte(config.Secret))
		_, _ = mac.Write([]byte(requestUri))
		hash = hex.EncodeToString(mac.Sum(nil))
	case "md5":
		hash = tools.Md5(fmt.Sprintf("%s%s", requestUri, config.Secret))
	default:
		hash = tools.Md5(fmt.Sprintf("%s%s", requestUri, config.Secret))
	}

	if hashLength > len(hash) {
		return hash
	}
	return hash[:hashLength]
}
