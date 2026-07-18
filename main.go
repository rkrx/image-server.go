package main

import (
	"github.com/gorilla/mux"
	"image-server/src/server"
	"image-server/src/tools"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	conf, err := tools.LoadConfig("config.toml")
	if err != nil {
		log.Fatalf("Can't load config.toml: %s", err)
	}

	if conf.LoggingEnabled {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(io.Discard)
	}

	log.Print("Start server...")

	log.Printf("Original backend: %s", conf.Server.OriginalBackend)
	log.Printf("Logging enabled: %t", conf.LoggingEnabled)

	backend := server.NewBackend(conf)
	r := mux.NewRouter()
	r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})
	r.PathPrefix("/").Handler(http.StripPrefix("/", backend))
	http.Handle("/", r)

	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	srv := &http.Server{
		Addr:              conf.Server.ListenAddr,
		Handler:           r,
		ReadHeaderTimeout: time.Duration(conf.Server.ReadHeaderTimeoutInSeconds) * time.Second,
		ReadTimeout:       time.Duration(conf.Server.ReadTimeoutInSeconds) * time.Second,
		WriteTimeout:      time.Duration(conf.Server.WriteTimeoutInSeconds) * time.Second,
		IdleTimeout:       time.Duration(conf.Server.IdleTimeoutInSeconds) * time.Second,
		MaxHeaderBytes:    conf.Server.MaxHeaderBytes,
		Protocols:         protocols,
		HTTP2: &http.HTTP2Config{
			MaxConcurrentStreams: conf.Server.MaxConcurrentHTTP2Streams,
			SendPingTimeout:      time.Duration(conf.Server.ReadHeaderTimeoutInSeconds) * time.Second,
			PingTimeout:          time.Duration(conf.Server.ReadHeaderTimeoutInSeconds) * time.Second,
			WriteByteTimeout:     time.Duration(conf.Server.WriteTimeoutInSeconds) * time.Second,
		},
	}

	err = srv.ListenAndServe()
	log.Fatal(err)
}
