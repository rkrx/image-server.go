package server

import (
	"fmt"
	"log"
	"reflect"
	"testing"
)

func TestParseRequestPath(t *testing.T) {
	input := "/ac15,3/rp500,auto/sh0.6/hs07AF3A88/camera.jpeg/webp"
	filename, commands, err := ParseRequestPath(input)
	fmt.Printf("Filename %s", filename)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Printf("Commands: %d\n", len(commands))
		for _, command := range commands {
			fmt.Printf("%s\n", "A command")
			fmt.Println(reflect.TypeOf(command))
		}
	}
}
