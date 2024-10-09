package server

import (
	"errors"
	"fmt"
	"image"
	"image-server/src/imaging"
	"image-server/src/tools"
	"log"
	"strings"
)

type ImageModificationCommand interface {
	apply(img image.Image) image.Image
}

type Autocrop struct {
	tolerance int
	border    int
}

func (cmd *Autocrop) apply(img image.Image) image.Image {
	return imaging.Autocrop(img, 15, float32(cmd.border))
}

type ResizeProportional struct {
	width  int
	height int
}

func (cmd *ResizeProportional) apply(img image.Image) image.Image {
	if cmd.width == -1 {
		cmd.width = img.Bounds().Dx()
	}
	if cmd.height == -1 {
		cmd.height = img.Bounds().Dy()
	}
	return imaging.ResizeProportional(img, cmd.width, cmd.height, imaging.Contain)
}

// Example: /ac15,3/rp500-auto/sh0.6/hsABCDEFG/camera.jpeg/webp
func ParseRequestPath(path string) (filename string, commands []ImageModificationCommand, err error) {
	path = strings.TrimLeft(path, "/")
	if success, groups := tools.RegexMatch(path, "^(.*?)/([\\w-]+\\.\\w+)/(\\w+)$"); success {
		//filename
		commands, err = parseCommands(groups[1])
		return
	}
	return
}

func parseCommands(commandStr string) ([]ImageModificationCommand, error) {
	parts := tools.RegexSplit(commandStr, "/+", -1)
	var commands []ImageModificationCommand
	for index := range parts {
		part := parts[index]
		if result, groups := tools.RegexMatch(part, "^ac([0-9]+),([0-9]+)$"); result {
			tolerance := tools.ToIntOr(groups[1], 0)
			border := tools.ToIntOr(groups[2], 0)
			commands = append(commands, &Autocrop{tolerance, border})
			log.Printf("autocrop(tolerance = %d, border = %d)", tolerance, border)
		} else if result, groups := tools.RegexMatch(part, "^rp([0-9]+|auto),([0-9]+|auto)$"); result {
			width := tools.ToIntOr(groups[1], -1)
			height := tools.ToIntOr(groups[2], -1)
			commands = append(commands, &ResizeProportional{width, height})
			log.Printf("resize-proportional(width = %d, height = %d)", width, height)
		} else if result, groups := tools.RegexMatch(part, "^sh((?:[0-9]*(?:\\.))?[0-9]+)$"); result {
			amount := tools.ToFloat32Or(groups[1], 0.0)
			log.Printf("sharpen(amount = %.2f)", amount)
		} else if result, groups := tools.RegexMatch(part, "^hs([0-9ABCDEF]{8})$"); result {
			hash := groups[1]
			log.Printf("hash(string = %s)", hash)
		} else {
			return []ImageModificationCommand{}, errors.New(fmt.Sprintf("unrecognized command pattern: %s", part))
		}
	}
	return commands, nil
}
