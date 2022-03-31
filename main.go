package main

import (
	"github.com/BurntSushi/toml"
	"github.com/chai2010/webp"
	"github.com/gorilla/mux"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	_ "golang.org/x/net/http2/h2c"
	"image-server/imaging"
	"image-server/server"
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
	if r.URL.Path == "/favicon.ico" {
		w.WriteHeader(404)
		return
	}

	log.Printf("RequestURI: %s; Path: %s; Method: %s", r.RequestURI, r.URL.Path, r.Method)

	// parts, err :=
	server.ParseRequestPath(r.URL.Path)

	w.Header().Add("Cache-Control", "no-cache")

	acceptedFormats := strings.Split(r.Header.Get("Accept"), ",")
	log.Printf("Accept: %s", acceptedFormats)

	useWebP := tools.Contains(acceptedFormats, "image/webp")
	useWebP = false

	img, err := imaging.LoadImageFromFile("var/images/camera.jpeg")
	if err != nil {
		w.WriteHeader(500)
		log.Println(err)
		return
	}

	width, height := 500, 500

	newImg := img
	newImg = imaging.Autocrop(img, 15, 3)
	newImg = imaging.ResizeProportional(newImg, width, height)
	newImg = imaging.Sharpen(newImg, .6)
	newImg = imaging.SetCanvas(newImg, width, height, imaging.Center, imaging.Center, color.White)

	var filename string
	if useWebP {
		filename = "var/cache/resized.webp"
		if err := imaging.SaveToWebpFile(newImg, filename, &webp.Options{Quality: 90}); err != nil {
			log.Println(err)
			return
		}
	} else {
		filename = "var/cache/resized.jpg"
		if err := imaging.SaveToJpegFile(newImg, filename, 90); err != nil {
			log.Println(err)
			return
		}
	}
	if err := ServeFromFile(filename, w); err != nil {
		log.Println(err)
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

type Config struct {
	OriginalBackend string
	LoggingEnabled  bool
	ListenAddr      string
}

func main() {
	log.Print("Start server...")

	var conf Config
	_, err := toml.DecodeFile("config.toml", &conf)
	if err != nil {
		log.Fatalf("Can't load config.toml: %s", err)
	}

	log.SetOutput(os.Stdout)

	log.Printf("Original backend: %s", conf.OriginalBackend)
	log.Printf("Logging enabled: %d", conf.LoggingEnabled)

	backend := server.Backend{BaseUri: conf.OriginalBackend, Logging: conf.LoggingEnabled}
	r := mux.NewRouter()
	r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})
	r.PathPrefix("/").Handler(http.StripPrefix("/", backend))
	http.Handle("/", r)

	h2s := &http2.Server{}

	srv := &http.Server{
		Addr:    conf.ListenAddr,
		Handler: h2c.NewHandler(r, h2s),
	}

	err = srv.ListenAndServe()
	log.Fatal(err)
}
