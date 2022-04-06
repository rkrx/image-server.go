package server

import (
	"fmt"
	"image-server/src/tools"
	"net/url"
)

func extractHashFromRequestUri(requestUri string, values url.Values, secret string) bool {
	if values.Has("hash") {
		var hash string
		success, groups := tools.RegexMatch(requestUri, "\\?.*?&hash=([a-f0-9]+)(&|$)")
		if success {
			hash = groups[1]

			requestUriWoHash := tools.RegexReplace(requestUri, "(\\?.*?)(&hash=(?:[a-f0-9]+))(&|$)", "$1$3")
			expectedHash := tools.Md5(fmt.Sprintf("%s%s", requestUriWoHash, secret))[:8]
			if expectedHash == hash {
				return true
				//} else {
				//	log.Printf("Wrong hash. Expected %s, got %s", expectedHash, hash)
			}
		}
	}
	return false
}
