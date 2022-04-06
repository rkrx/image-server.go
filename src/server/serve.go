package server

import (
	"fmt"
	"image"
	"image-server/src/imaging"
	"image-server/src/tools"
	"image/color"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

type Backend struct {
	Config tools.Config
}

func (backend Backend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if backend.Config.DisableCaching {
		w.Header().Add("Cache-Control", "no-cache")
	}

	profile := backend.Config.GetDefaultProfile()

	//region Check if the url has a profile key
	if r.URL.Query().Has("profile") {
		profileName := r.URL.Query().Get("profile")
		var err error
		if profile, err = backend.Config.GetProfile(profileName); err != nil {
			w.WriteHeader(404)
			return
		}
	}
	//endregion

	//region Check if the client ip found in DisableHashCheckingFromIPv4Sources
	getRemoteIP := func(header http.Header, remoteAddrHeader string) string {
		if remoteAddrHeader != "" {
			if ip := r.Header.Get(remoteAddrHeader); ip != "" {
				if success, _ := tools.RegexMatch(ip, "^(?:\\[[(:0-9a-fA-F)]+\\]|(\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3})):\\d+$"); success {
					ipAddr, _, err := net.SplitHostPort(ip)
					if err != nil {
						return ""
					}
					return net.ParseIP(ipAddr).String()
				}
				return net.ParseIP(ip).String()
			}
		} else {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err == nil {
				return net.ParseIP(ip).String()
			}
		}
		return ""
	}
	remoteAddrIp := getRemoteIP(r.Header, backend.Config.Server.RemoteAddrHeader)
	if remoteAddrIp != "" && tools.Contains(backend.Config.DisableHashCheckingFromIPv4Sources, remoteAddrIp) {
		backend.Config.DisableHashChecking = true
	}
	//endregion

	//region Apply URL-Parameters to the current profile
	if err := applyUrlParameters(&profile, r.RequestURI, r.URL.Query(), backend.Config.Secret, backend.Config.DisableHashChecking); err != nil {
		w.WriteHeader(403)
		return
	}
	//endregion

	//region Check if the URL has a output file type
	if r.URL.Query().Has("type") {
		fileType := strings.ToLower(r.URL.Query().Get("type"))
		if err := profile.ApplyFileType(fileType); err != nil {
			w.WriteHeader(403)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
	}
	//endregion

	//region Project the origFilename and the cachePath
	origFilename, cachePath := "", ""
	if backend.Config.HashCacheFilenames {
		origFilename = tools.Md5(fmt.Sprintf("%s/%s", profile.GetDirectoryHash(), r.URL.Path))
		cachePath = fmt.Sprintf("%s/%s.%s", profile.ImageCacheDir, origFilename, profile.Type)
	} else {
		origFilename = tools.Md5(r.URL.Path)
		cachePath = fmt.Sprintf("%s/%s/%s.%s", profile.ImageCacheDir, profile.GetDirectoryHash(), origFilename, profile.Type)
	}

	if success, groups := tools.RegexMatch(r.URL.Path, "\\.(jpe?g|gif|png|webp)$"); success {
		origFilename = fmt.Sprintf("%s.%s", origFilename, groups[1])
	}
	//endregion

	//region Generate the cache image
	if !tools.FileExists(cachePath) {
		log.Printf("File not in cache: %s", cachePath)
		statusCode, err := backend.getFinalImageBytes(r.URL.Path, origFilename, cachePath, profile, func(img image.Image) image.Image {
			if profile.Width < 1 && profile.Height < 1 {
				profile.Width = img.Bounds().Dx()
				profile.Height = img.Bounds().Dy()
			}
			img = imaging.Autocrop(img, profile.Autocrop.Tolerance, profile.Autocrop.Border)
			img = imaging.ResizeProportional(img, profile.Width, profile.Height)
			img = imaging.Sharpen(img, float64(profile.Sharpen))
			img = imaging.SetCanvas(img, profile.Width, profile.Height, imaging.Center, imaging.Center, color.White)
			return img
		})

		if err != nil {
			log.Print(err)
			w.WriteHeader(statusCode)
			return
		}
	} else {
		log.Printf("File in cache: %s", cachePath)
	}
	//endregion

	//region Write the cached image contents to the client result
	file, err := os.Open(cachePath)
	if err != nil {
		log.Printf("Could not open file %s: %s", cachePath, err)
		w.WriteHeader(500)
		return
	}
	defer file.Close()
	_, err = io.Copy(w, file)
	if err != nil {
		log.Println(err)
		return
	}
	//endregion
}
