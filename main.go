package main

import (
	"bytes"
	"fmt"
	"github.com/nfnt/resize"
	"image"
	"image-server/imaging"
	"image-server/tools"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

func onRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("Method: %s", r.Method)

	acceptedFormats := strings.Split(r.Header.Get("Accept"), ",")
	log.Printf("Accept: %s", acceptedFormats)

	useWebP := tools.Contains(acceptedFormats, "image/webp")
	useWebP = false

	if useWebP {
		w.Header().Add("Content-Type", "image/webp")
	} else {
		w.Header().Add("Content-Type", "image/jpeg")
	}
	//for name, values := range r.Header {
	//	log.Printf("Headers: %s %s", name, values)
	//}

	var buf bytes.Buffer
	img, err := imaging.LoadImageFromFile("img.png")
	if err != nil {
		log.Fatal(err)
	}

	resizedImg := resize.Resize(100, 100, img, resize.Lanczos3)

	if useWebP {
		if err := imaging.Img2WebpBuffer(&buf, resizedImg, 90); err != nil {
			log.Fatal(err)
			return
		}
	} else {
		if err := imaging.Img2JpegBuffer(&buf, resizedImg, 90); err != nil {
			log.Fatal(err)
			return
		}
	}

	log.Printf("%d", buf.Len())

	if err != nil {
		log.Fatal(err)
		return
	}

	w.Write(buf.Bytes())
	if err = ioutil.WriteFile("cache/resized.webp", buf.Bytes(), 0666); err != nil {
		log.Fatal(err)
	}
}

func doWork(img image.Image) {
	w, h := 500, 500
	newImg := img
	newImg = imaging.Autocrop(img, 85, 100, 3)
	newImg = imaging.ResizeProportional(newImg, w, h)
	newImg = imaging.Sharpen(newImg, 1)
	newImg = imaging.SetCanvas(newImg, w, h, imaging.Center, imaging.Center, color.White)
}

func worker(caption string, c chan int) {
	img, err := imaging.LoadImageFromFile("camera.jpeg")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Start worker %s\n", caption)

	for {
		index := <-c
		start := time.Now()
		doWork(img)
		elapsed := time.Since(start)
		fmt.Printf("%s %d took %s\n", caption, index, elapsed)
	}
}

func main() {
	//img, err := imaging.LoadImageFromFile("camera.jpeg")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//w, h := 500, 500
	//
	//newImg := imaging.Autocrop(img, 85, 100, 3)
	//newImg = imaging.ResizeProportional(newImg, w, h)
	//newImg = imaging.Sharpen(newImg, 1)
	//newImg = imaging.SetCanvas(newImg, w, h, imaging.Center, imaging.Center, color.White)
	//
	//webp.Save("test.webp", newImg, &webp.Options{Quality: 100})
	//
	//return

	c := make(chan int)

	start := time.Now()

	go worker("A", c)
	go worker("B", c)
	go worker("C", c)
	go worker("D", c)
	go worker("E", c)
	go worker("F", c)
	go worker("G", c)
	go worker("G", c)
	go worker("I", c)
	go worker("J", c)

	for i := 0; i < 1000; i++ {
		c <- i
	}

	elapsed := time.Since(start)
	fmt.Printf("Execution took %s\n", elapsed)

	//webp.Save("test.webp", newImg, &webp.Options{Quality: 100.0})

	//http.HandleFunc("/", onRequest)
	//err := http.ListenAndServe("127.0.0.1:8080", nil)
	//log.Fatal(err)
}
