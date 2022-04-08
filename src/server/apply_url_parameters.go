package server

import (
	"errors"
	"image-server/src/tools"
	"net/url"
)

func applyUrlParameters(profile *tools.Profile, requestUri string, values url.Values, secret string, diableHashChecking bool) error {
	var tryApplyUrlParameter = func(key string, fn func(value string) bool) func() bool {
		return func() bool {
			if values.Has(key) {
				value := values.Get(key)
				return fn(value)
			}
			return false
		}
	}

	params := []func() bool{
		tryApplyUrlParameter("width", func(value string) bool {
			profile.Width = tools.ToIntOr(value, profile.Width)
			return true
		}),
		tryApplyUrlParameter("height", func(value string) bool {
			profile.Height = tools.ToIntOr(value, profile.Height)
			return true
		}),
		tryApplyUrlParameter("sharpen", func(value string) bool {
			profile.Sharpen = tools.ToFloat32Or(value, profile.Sharpen)
			return true
		}),
		tryApplyUrlParameter("autocrop", func(value string) bool {
			if success, groups := tools.RegexMatch(value, "^(\\d+)(?:,(\\d+))?$"); success {
				tolerance := tools.ToFloat32Or(groups[1], -1)
				border := tools.ToFloat32Or(groups[2], 0)
				profile.Autocrop.Apply = true
				profile.Autocrop.Tolerance = tolerance
				profile.Autocrop.Border = border
			}
			return true
		}),
	}

	parameterFound := false
	for _, fn := range params {
		if fn() {
			parameterFound = true
		}
	}

	if parameterFound && !diableHashChecking && !extractHashFromRequestUri(requestUri, values, secret) {
		return errors.New("hash value failure")
	}

	return nil
}
