package main

import (
	"fmt"
	"image"
	"image-server/imaging"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"time"
)

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
