package server

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image-server/src/imaging"
	"image-server/src/tools"
	"net/url"
	"strings"
	"unicode/utf8"
)

type compositionSource struct {
	Path    string
	Profile tools.Profile
}

type compositionPlan struct {
	nodes     []*imaging.CompositionNode
	sources   map[string]compositionSource
	cacheHash string
}

// compositionPayload resolves the two mutually exclusive transport formats.
// Decoding happens after request limits and signature checks, before JSON validation.
func compositionPayload(values url.Values) (string, error) {
	if values.Has("put") {
		if values.Has("putb") || len(values["put"]) != 1 {
			return "", errors.New("provide exactly one put or putb parameter")
		}
		return values.Get("put"), nil
	}
	if len(values["putb"]) != 1 || strings.ContainsAny(values.Get("putb"), " \t\r\n") {
		return "", errors.New("putb requires one Base64 value without whitespace")
	}
	for _, encoding := range []*base64.Encoding{
		base64.StdEncoding.Strict(), base64.RawStdEncoding.Strict(),
		base64.URLEncoding.Strict(), base64.RawURLEncoding.Strict(),
	} {
		decoded, err := encoding.DecodeString(values.Get("putb"))
		if err == nil && utf8.Valid(decoded) {
			return string(decoded), nil
		}
	}
	return "", errors.New("putb must contain Base64-encoded UTF-8 JSON")
}

type compositionBudget struct {
	sourceRemaining, renderRemaining int64
}

func newCompositionBudget(config tools.Config) *compositionBudget {
	return &compositionBudget{
		sourceRemaining: positiveInt64OrDefault(config.MaxCompositionSourcePixels, tools.DefaultMaxCompositionSourcePixels),
		renderRemaining: positiveInt64OrDefault(config.MaxCompositionRenderPixels, tools.DefaultMaxCompositionRenderPixels),
	}
}

func positiveInt64OrDefault(value, fallback int64) int64 {
	if value > 0 {
		return value
	}
	return fallback
}

func chargeCompositionPixels(remaining *int64, width, height int) error {
	if width < 1 || height < 1 || int64(width) > *remaining/int64(height) {
		return imaging.ErrCompositionLimit
	}
	*remaining -= int64(width) * int64(height)
	return nil
}

func (b *compositionBudget) chargeSource(width, height int) error {
	if b == nil {
		return nil
	}
	return chargeCompositionPixels(&b.sourceRemaining, width, height)
}

func (b *compositionBudget) chargeRender(width, height int) error {
	if b == nil {
		return nil
	}
	return chargeCompositionPixels(&b.renderRemaining, width, height)
}

func (backend *Backend) prepareComposition(value string, root tools.Profile) (*compositionPlan, error) {
	nodes, err := imaging.ParseComposition(value, imaging.CompositionLimits{
		MaxNodes:  positiveOrDefault(backend.Config.MaxCompositionNodes, tools.DefaultMaxCompositionNodes),
		MaxImages: positiveOrDefault(backend.Config.MaxCompositionImages, tools.DefaultMaxCompositionImages),
		MaxDepth:  positiveOrDefault(backend.Config.MaxCompositionDepth, tools.DefaultMaxCompositionDepth),
	})
	if err != nil {
		return nil, err
	}
	plan := &compositionPlan{nodes: nodes, sources: make(map[string]compositionSource)}
	var sourceKeys []compositionSource
	var resolve func([]*imaging.CompositionNode) error
	resolve = func(elements []*imaging.CompositionNode) error {
		for _, node := range elements {
			if node.Type == "rect" {
				if err := resolve(node.Children); err != nil {
					return err
				}
				continue
			}
			if _, exists := plan.sources[node.URL]; exists {
				continue
			}
			source, err := backend.resolveCompositionSource(node.URL)
			if err != nil {
				return err
			}
			plan.sources[node.URL] = source
			sourceKeys = append(sourceKeys, source)
		}
		return nil
	}
	if err := resolve(nodes); err != nil {
		return nil, err
	}
	// Include effective profiles (including child origins) and processing limits.
	// Changing a profile or tightening limits must not reuse a stale derivative.
	key, err := json.Marshal(struct {
		Version int
		Nodes   []*imaging.CompositionNode
		Root    tools.Profile
		Sources []compositionSource
		Limits  []int64
	}{1, nodes, root, sourceKeys, []int64{
		backend.Config.OriginMaxImageBytes,
		int64(backend.Config.MaxSourceImageWidth), int64(backend.Config.MaxSourceImageHeight), backend.Config.MaxSourceImagePixels,
		int64(backend.Config.MaxOutputImageWidth), int64(backend.Config.MaxOutputImageHeight), backend.Config.MaxOutputImagePixels,
		int64(backend.Config.MaxCompositionNodes), int64(backend.Config.MaxCompositionImages), int64(backend.Config.MaxCompositionDepth),
		backend.Config.MaxCompositionSourcePixels, backend.Config.MaxCompositionRenderPixels,
	}})
	if err != nil {
		return nil, err
	}
	plan.cacheHash = fmt.Sprintf("%x", sha256.Sum256(key))
	return plan, nil
}

func (backend *Backend) resolveCompositionSource(value string) (compositionSource, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.User != nil || parsed.Opaque != "" || parsed.Fragment != "" || !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(parsed.Path, "//") || strings.Contains(parsed.Path, "\\") {
		return compositionSource{}, errors.New("child URL must be an origin-relative image path")
	}
	for _, part := range strings.Split(parsed.Path, "/") {
		if part == "." || part == ".." {
			return compositionSource{}, errors.New("child URL cannot traverse origin directories")
		}
	}
	values, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		return compositionSource{}, errors.New("invalid child query")
	}
	for name, entries := range values {
		if len(entries) != 1 {
			return compositionSource{}, errors.New("duplicate child parameter")
		}
		switch name {
		case "profile", "width", "height", "mode", "sharpen", "autocrop", "type", "hash":
		default:
			return compositionSource{}, errors.New("unsupported child parameter")
		}
	}
	profile := backend.Config.GetDefaultProfile()
	// The signed outer URI already authorizes the complete child operation.
	if err := applyUrlParameters(&profile, backend.Config, value, values, true); err != nil {
		return compositionSource{}, err
	}
	if values.Has("type") {
		if err := profile.ApplyFileType(values.Get("type")); err != nil {
			return compositionSource{}, err
		}
	}
	return compositionSource{Path: parsed.EscapedPath(), Profile: profile}, nil
}

func (backend *Backend) composeImage(ctx context.Context, background image.Image, plan *compositionPlan, budget *compositionBudget) (image.Image, error) {
	return imaging.Compose(ctx, background, plan.nodes, func(ctx context.Context, childURL string) (image.Image, error) {
		source := plan.sources[childURL]
		originURL := tools.ConcatPathParts(source.Profile.OriginalBackend, source.Path)
		filename := fmt.Sprintf("source-%x", sha256.Sum256([]byte(originURL)))
		img, status, err := backend.loadSourceImage(ctx, originURL, filename, source.Profile, budget)
		if err != nil {
			return nil, &imageHTTPError{status, err}
		}
		return backend.transformImage(img, source.Profile, true, budget)
	}, func(w, h int) error {
		if err := backend.Config.ValidateOutputImageDimensions(w, h); err != nil {
			return err
		}
		return budget.chargeRender(w, h)
	})
}
