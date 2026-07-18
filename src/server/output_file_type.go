package server

import (
	"mime"
	"net/http"
	"strconv"
	"strings"

	"image-server/src/tools"
)

func applyOutputFileType(profile *tools.Profile, request *http.Request) (bool, error) {
	if request.URL.Query().Has("type") {
		return false, profile.ApplyFileType(request.URL.Query().Get("type"))
	}

	profile.Type = negotiateOutputFileType(request.Header.Get("Accept"))
	return true, nil
}

func negotiateOutputFileType(acceptHeader string) tools.FileType {
	avifQuality := acceptedMediaTypeQuality(acceptHeader, "image/avif")
	webpQuality := acceptedMediaTypeQuality(acceptHeader, "image/webp")

	if avifQuality > 0 && avifQuality >= webpQuality {
		return tools.Avif
	}
	if webpQuality > 0 {
		return tools.WebP
	}
	return tools.Jpeg
}

func acceptedMediaTypeQuality(acceptHeader, expectedMediaType string) float64 {
	bestQuality := float64(0)
	for _, acceptValue := range strings.Split(acceptHeader, ",") {
		mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(acceptValue))
		if err != nil || mediaType != expectedMediaType {
			continue
		}

		quality := float64(1)
		if qualityValue, ok := params["q"]; ok {
			quality, err = strconv.ParseFloat(qualityValue, 64)
			if err != nil || quality < 0 || quality > 1 {
				continue
			}
		}
		if quality > bestQuality {
			bestQuality = quality
		}
	}
	return bestQuality
}
