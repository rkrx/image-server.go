package main

import (
	"github.com/gorilla/mux"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	_ "golang.org/x/net/http2/h2c"
	"image-server/server"
	"image-server/tools"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"os"
)

func ServeFromFile(filepath string, writer http.ResponseWriter) error {
	bytes, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}
	_, err = writer.Write(bytes)
	return err
}

func main() {
	log.Print("Start server...")

	conf, err := tools.LoadConfig("config.toml")
	if err != nil {
		log.Fatalf("Can't load config.toml: %s", err)
	}

	log.SetOutput(os.Stdout)

	log.Printf("Original backend: %s", conf.Server.OriginalBackend)
	log.Printf("Logging enabled: %t", conf.LoggingEnabled)

	backend := server.Backend{Config: conf}
	r := mux.NewRouter()
	r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})
	r.PathPrefix("/").Handler(http.StripPrefix("/", backend))
	http.Handle("/", r)

	h2s := &http2.Server{}

	srv := &http.Server{
		Addr:    conf.Server.ListenAddr,
		Handler: h2c.NewHandler(r, h2s),
	}

	err = srv.ListenAndServe()
	log.Fatal(err)
}
