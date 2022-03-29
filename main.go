package main

import (
	"github.com/chai2010/webp"
	"image-server/imaging"
	"image-server/tools"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"os"
	"strings"
)

func onRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("RequestURI: %s", r.RequestURI)
	log.Printf("Path: %s", r.URL.Path)
	log.Printf("Method: %s", r.Method)

	w.Header().Add("Cache-Control", "no-cache")

	acceptedFormats := strings.Split(r.Header.Get("Accept"), ",")
	log.Printf("Accept: %s", acceptedFormats)

	useWebP := tools.Contains(acceptedFormats, "image/webp")
	useWebP = false

	img, err := imaging.LoadImageFromFile("camera.jpeg")
	if err != nil {
		log.Fatal(err)
	}

	width, height := 500, 500

	newImg := img
	newImg = imaging.Autocrop(img, 85, 100, 3)
	newImg = imaging.ResizeProportional(newImg, width, height)
	newImg = imaging.Sharpen(newImg, .6)
	newImg = imaging.SetCanvas(newImg, width, height, imaging.Center, imaging.Center, color.White)

	var filename string
	if useWebP {
		filename = "cache/resized.webp"
		if err := imaging.SaveToWebpFile(newImg, filename, &webp.Options{Quality: 90}); err != nil {
			log.Fatal(err)
			return
		}
	} else {
		filename = "cache/resized.jpg"
		if err := imaging.SaveToJpegFile(newImg, filename, 90); err != nil {
			log.Fatal(err)
			return
		}
	}
	if err := ServeFromFile(filename, w); err != nil {
		log.Fatal(err)
		return
	}
}

func ServeFromFile(filepath string, writer http.ResponseWriter) error {
	bytes, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}
	_, err = writer.Write(bytes)
	return err
}

func main() {
	http.HandleFunc("/", onRequest)
	err := http.ListenAndServe("127.0.0.1:8080", nil)
	log.Fatal(err)
}
