package main

import (
	"fmt"
	"image"
	imaging2 "image-server/src/imaging"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"time"
)

func doWork(img image.Image) image.Image {
	w, h := 500, 500
	newImg := img
	newImg = imaging2.Autocrop(img, 15, 3)
	newImg = imaging2.ResizeProportional(newImg, w, h, imaging2.Contain)
	newImg = imaging2.Sharpen(newImg, 1)
	newImg = imaging2.SetCanvas(newImg, w, h, imaging2.Center, imaging2.Center, color.White)
	return newImg
}

func worker(caption string, c chan int) {
	img, err := imaging2.LoadImageFromFile("camera.jpeg")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Start worker %s\n", caption)

	for {
		index := <-c
		start := time.Now()
		_ = doWork(img)
		elapsed := time.Since(start)
		fmt.Printf("%s %d took %s\n", caption, index, elapsed)
	}
}

func main() {
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
}
