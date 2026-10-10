package server

import (
	"errors"
	"fmt"
	"image-server/src/tools"
	"math"
	"net/url"
	"strconv"
)

var errInvalidRequestHash = errors.New("invalid request hash")

func applyUrlParameters(profile *tools.Profile, config tools.Config, requestUri string, values url.Values, disableHashChecking bool) error {
	if values.Has("profile") {
		namedProfile, err := config.GetProfile(values.Get("profile"))
		if err != nil {
			return errors.New("unknown image profile")
		}
		profile.ApplyFromProfile(namedProfile)
	}

	// A composition is a transformation even when no other parameters are set.
	parameterFound := values.Has("put") || values.Has("putb") || values.Has("canvas")
	if values.Has("width") {
		width, err := parseIntegerParameter("width", values.Get("width"))
		if err != nil {
			return err
		}
		profile.Width = width
		parameterFound = true
	}
	if values.Has("height") {
		height, err := parseIntegerParameter("height", values.Get("height"))
		if err != nil {
			return err
		}
		profile.Height = height
		parameterFound = true
	}
	if values.Has("mode") {
		mode := values.Get("mode")
		if mode != "proportional" && mode != "contain" && mode != "cover" && mode != "stretch" {
			return errors.New("invalid resize mode")
		}
		profile.Mode = mode
		parameterFound = true
	}
	if values.Has("sharpen") {
		sharpen, err := strconv.ParseFloat(values.Get("sharpen"), 32)
		if err != nil || math.IsNaN(sharpen) || math.IsInf(sharpen, 0) || sharpen < 0 || sharpen > 10 {
			return errors.New("sharpen must be between 0 and 10")
		}
		profile.Sharpen = float32(sharpen)
		parameterFound = true
	}
	if values.Has("autocrop") {
		success, groups := tools.RegexMatch(values.Get("autocrop"), "^(\\d+)(?:,(\\d+))?$")
		if !success {
			return errors.New("invalid autocrop parameter")
		}
		tolerance := tools.ToFloat32Or(groups[1], -1)
		border := tools.ToFloat32Or(groups[2], 0)
		if tolerance < 0 || tolerance > 100 || border < 0 || border > 25 {
			return errors.New("autocrop tolerance must be between 0 and 100 and border between 0 and 25")
		}
		profile.Autocrop.Apply = true
		profile.Autocrop.Tolerance = tolerance
		profile.Autocrop.Border = border
		parameterFound = true
	}

	if err := config.ValidateRequestedOutputDimensions(profile.Width, profile.Height); err != nil {
		return err
	}

	if parameterFound && !disableHashChecking && !extractHashFromRequestUri(requestUri, values, config) {
		return errInvalidRequestHash
	}

	return nil
}

func parseIntegerParameter(name, value string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	return parsed, nil
}
