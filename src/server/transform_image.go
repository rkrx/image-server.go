package server

import (
	"image"
	"image-server/src/imaging"
	"image-server/src/tools"
	"image/color"
)

// transformImage is shared by the background and children. Children start from
// their own profile, and composition backgrounds defer sharpening until the end.
func (backend *Backend) transformImage(img image.Image, profile tools.Profile, sharpen bool, budget *compositionBudget) (image.Image, error) {
	if profile.Width < 1 && profile.Height < 1 {
		profile.Width, profile.Height = img.Bounds().Dx(), img.Bounds().Dy()
	}
	if profile.Autocrop.Apply {
		var err error
		img, err = imaging.AutocropChecked(img, profile.Autocrop.Tolerance, profile.Autocrop.Border, func(w, h int) error {
			if err := backend.Config.ValidateSourceImageDimensions(w, h); err != nil {
				return err
			}
			return budget.chargeRender(w, h)
		})
		if err != nil {
			return nil, err
		}
	}
	width, height := profile.Width, profile.Height
	switch profile.Mode {
	case "cover":
		width, height = imaging.CalcProportionalOuterMeasures(img.Bounds().Dx(), img.Bounds().Dy(), width, height)
	case "proportional", "contain":
		width, height = imaging.CalcProportionalInnerMeasures(img.Bounds().Dx(), img.Bounds().Dy(), width, height)
	case "stretch":
		if width == 0 || height == 0 {
			width, height = imaging.CalcProportionalInnerMeasures(img.Bounds().Dx(), img.Bounds().Dy(), width, height)
		}
	}
	if err := backend.Config.ValidateOutputImageDimensions(width, height); err != nil {
		return nil, err
	}
	canvasWidth, canvasHeight := profile.Width, profile.Height
	if canvasWidth == 0 {
		canvasWidth = width
	}
	if canvasHeight == 0 {
		canvasHeight = height
	}
	if profile.Mode == "contain" || profile.Mode == "cover" {
		if err := backend.Config.ValidateOutputImageDimensions(canvasWidth, canvasHeight); err != nil {
			return nil, err
		}
	}
	if err := budget.chargeRender(width, height); err != nil {
		return nil, err
	}
	img = imaging.Resize(img, width, height)
	if sharpen && profile.Sharpen > 0 {
		if err := budget.chargeRender(width, height); err != nil {
			return nil, err
		}
		img = imaging.Sharpen(img, float64(profile.Sharpen))
	}
	if profile.Mode == "contain" || profile.Mode == "cover" {
		if err := budget.chargeRender(canvasWidth, canvasHeight); err != nil {
			return nil, err
		}
		img = imaging.SetCanvas(img, canvasWidth, canvasHeight, imaging.Center, imaging.Center, color.White)
	}
	return img, nil
}
