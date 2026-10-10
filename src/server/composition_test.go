package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image-server/src/imaging"
	"image-server/src/tools"
	"image/color"
	"image/draw"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func compositionTestPNG(w, h int, c color.Color) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(c), image.Point{}, draw.Src)
	var output bytes.Buffer
	if err := png.Encode(&output, img); err != nil {
		panic(err)
	}
	return output.Bytes()
}

func compositionTestBackend(t *testing.T) (*Backend, *atomic.Int32) {
	t.Helper()
	var requests atomic.Int32
	background := compositionTestPNG(8, 8, color.White)
	child := compositionTestPNG(4, 2, color.NRGBA{R: 80, G: 80, B: 80, A: 255})
	border := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	draw.Draw(border, border.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(border, image.Rect(2, 3, 6, 5), image.NewUniform(color.NRGBA{R: 80, G: 80, B: 80, A: 255}), image.Point{}, draw.Src)
	var borderPNG bytes.Buffer
	require.NoError(t, png.Encode(&borderPNG, border))
	plus := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	for y := 0; y < 3; y++ {
		for x := 0; x < 3; x++ {
			if x == 1 || y == 1 {
				plus.SetNRGBA(x, y, color.NRGBA{R: 255, B: 128, A: 255})
			}
		}
	}
	var plusPNG bytes.Buffer
	require.NoError(t, png.Encode(&plusPNG, plus))
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		assert.Empty(t, r.URL.RawQuery, "transformation query must not reach the origin")
		switch r.URL.Path {
		case "/background.png":
			_, _ = w.Write(background)
		case "/child.png", "/has?mark.png":
			_, _ = w.Write(child)
		case "/border.png":
			_, _ = w.Write(borderPNG.Bytes())
		case "/plus.png":
			_, _ = w.Write(plusPNG.Bytes())
		case "/red.png":
			_, _ = w.Write(compositionTestPNG(4, 2, color.NRGBA{R: 255, A: 255}))
		case "/green.png":
			_, _ = w.Write(compositionTestPNG(1, 1, color.NRGBA{G: 255, A: 255}))
		case "/blue.png":
			_, _ = w.Write(compositionTestPNG(2, 4, color.NRGBA{B: 255, A: 255}))
		case "/corrupt.png":
			_, _ = w.Write([]byte("not an image"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(origin.Close)
	backend := NewBackend(tools.Config{
		DisableHashChecking: true,
		Secret:              "composition-test",
		HashAlgorithm:       "sha256",
		HashLength:          32,
		ImageCacheDir:       filepath.Join(t.TempDir(), "images"),
		Server: tools.ServerType{
			OriginalBackend:        origin.URL,
			ImageProcessingWorkers: 1,
		},
		JpegDefaults: tools.JpegDefaults{Quality: 85},
		WebpDefaults: tools.WebpDefaults{Quality: 80, Lossless: true},
	})
	return backend, &requests
}

func compositionTarget(scene string, extra url.Values) string {
	if extra == nil {
		extra = make(url.Values)
	}
	extra.Set("put", scene)
	if !extra.Has("type") {
		extra.Set("type", "png")
	}
	return "/background.png?" + extra.Encode()
}

func serveCompositionTest(backend *Backend, target string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	backend.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
	return response
}

func decodeCompositionTest(t *testing.T, response *httptest.ResponseRecorder) image.Image {
	t.Helper()
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	img, _, err := image.Decode(bytes.NewReader(response.Body.Bytes()))
	require.NoError(t, err)
	return img
}

const compositionTestScene = `[{"type":"rect","pos":[25,25],"dim":[50,50],"anc":"center center","cld":[{"type":"image","url":"/child.png","pos":[-50,-50],"dim":[100,100]}]}]`

func TestCompositionHTTPThreePlusPivotsShareTransportsAndRespectParentAnchor(t *testing.T) {
	for _, anchor := range []string{"left top", "center center"} {
		t.Run(anchor, func(t *testing.T) {
			backend, requests := compositionTestBackend(t)
			var children []map[string]any
			for _, y := range []int{25, 50, 75} {
				pos := []int{50, y}
				if anchor == "center center" {
					pos = []int{0, y - 50}
				}
				children = append(children, map[string]any{"type": "image", "url": "/plus.png", "pos": pos, "dim": []any{10, nil}, "piv": "center center", "mod": "pixelated"})
			}
			encodedScene, err := json.Marshal([]map[string]any{{"type": "rect", "pos": []int{0, 0}, "dim": []int{100, 100}, "anc": anchor, "cld": children}})
			require.NoError(t, err)
			scene := string(encodedScene)
			extra := url.Values{"canvas": {"white"}, "width": {"400"}, "height": {"400"}, "type": {"png"}}
			plain := serveCompositionTest(backend, compositionTarget(scene, extra))
			img := decodeCompositionTest(t, plain)
			assert.Equal(t, image.Rect(0, 0, 400, 400), img.Bounds())
			magenta := color.NRGBA{R: 255, B: 128, A: 255}
			white := color.NRGBAModel.Convert(color.White)
			for _, y := range []int{100, 200, 300} {
				for _, point := range []image.Point{{X: 200, Y: y}, {X: 180, Y: y}, {X: 219, Y: y}, {X: 200, Y: y - 20}, {X: 200, Y: y + 19}} {
					assert.Equal(t, magenta, color.NRGBAModel.Convert(img.At(point.X, point.Y)), point)
				}
				for _, point := range []image.Point{{X: 179, Y: y}, {X: 220, Y: y}, {X: 200, Y: y - 21}, {X: 200, Y: y + 20}} {
					assert.Equal(t, white, color.NRGBAModel.Convert(img.At(point.X, point.Y)), point)
				}
			}
			extra.Del("put")
			extra.Set("putb", base64.RawURLEncoding.EncodeToString(encodedScene))
			viaBase64 := serveCompositionTest(backend, "/background.png?"+extra.Encode())
			require.Equal(t, http.StatusOK, viaBase64.Code)
			assert.Equal(t, plain.Body.Bytes(), viaBase64.Body.Bytes())
			assert.Equal(t, int32(1), requests.Load())
		})
	}
}

func TestCompositionHTTPPivotChangesCacheAndRequiresSignature(t *testing.T) {
	for _, hashNames := range []bool{false, true} {
		t.Run(map[bool]string{false: "directories", true: "hashed filenames"}[hashNames], func(t *testing.T) {
			backend, requests := compositionTestBackend(t)
			backend.Config.HashCacheFilenames = hashNames
			scene := `[{"type":"image","url":"/child.png","pos":[50,50],"dim":[50,50],"fit":"stretch"}]`
			extra := url.Values{"canvas": {"white"}, "width": {"40"}, "height": {"40"}, "type": {"png"}}
			legacy := serveCompositionTest(backend, compositionTarget(scene, extra))
			decodeCompositionTest(t, legacy)
			centeredScene := strings.Replace(scene, `"type":"image"`, `"type":"image","piv":"center center"`, 1)
			centeredTarget := compositionTarget(centeredScene, extra)
			centered := serveCompositionTest(backend, centeredTarget)
			decodeCompositionTest(t, centered)
			assert.NotEqual(t, legacy.Body.Bytes(), centered.Body.Bytes())
			pattern := filepath.Join(backend.Config.ImageCacheDir, "composition-v1-*", "*.png")
			if hashNames {
				pattern = filepath.Join(backend.Config.ImageCacheDir, "*.png")
			}
			derivatives, err := filepath.Glob(pattern)
			require.NoError(t, err)
			require.Len(t, derivatives, 2)
			backend.Config.DisableHashChecking = false
			assert.Equal(t, http.StatusForbidden, serveCompositionTest(backend, centeredTarget).Code, "cached requests still require a signature")
			signed := centeredTarget + "&hash=" + generateRequestHash(centeredTarget, backend.Config)
			assert.Equal(t, http.StatusOK, serveCompositionTest(backend, signed).Code)
			tampered := strings.Replace(signed, url.QueryEscape(centeredScene), url.QueryEscape(scene), 1)
			assert.Equal(t, http.StatusForbidden, serveCompositionTest(backend, tampered).Code)
			assert.Equal(t, int32(1), requests.Load())
		})
	}
	backend, requests := compositionTestBackend(t)
	for _, typ := range []string{"rect", "image"} {
		scene := `[{"type":"` + typ + `","pos":[0,0],"dim":[10,10],"piv":"top left"` + map[string]string{"rect": "", "image": `,"url":"/child.png"`}[typ] + `}]`
		assert.Equal(t, http.StatusBadRequest, serveCompositionTest(backend, compositionTarget(scene, nil)).Code)
		encoded := base64.RawURLEncoding.EncodeToString([]byte(scene))
		assert.Equal(t, http.StatusBadRequest, serveCompositionTest(backend, "/background.png?putb="+encoded).Code)
	}
	assert.Zero(t, requests.Load())
}

func TestCompositionPayloadBase64VariantsPreserveUTF8(t *testing.T) {
	payload := `[{"url":"/þÿ"},{"url":"/ÿÿÿ"}]`
	standard := base64.StdEncoding.EncodeToString([]byte(payload))
	require.Contains(t, standard, "+")
	require.Contains(t, standard, "/")
	for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		encoded := encoding.EncodeToString([]byte(payload))
		decoded, err := compositionPayload(url.Values{"putb": {encoded}})
		require.NoError(t, err)
		assert.Equal(t, payload, decoded)
	}
}

func TestCompositionHTTPBase64SharesJSONCache(t *testing.T) {
	for _, hashNames := range []bool{false, true} {
		t.Run(map[bool]string{false: "directories", true: "hashed filenames"}[hashNames], func(t *testing.T) {
			backend, requests := compositionTestBackend(t)
			backend.Config.HashCacheFilenames = hashNames
			plain := serveCompositionTest(backend, compositionTarget(compositionTestScene, nil))
			decodeCompositionTest(t, plain)
			for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
				// JSON whitespace must not create another derivative either.
				encoded := encoding.EncodeToString([]byte(compositionTestScene + " "))
				target := "/background.png?" + url.Values{"putb": {encoded}, "type": {"png"}}.Encode()
				response := serveCompositionTest(backend, target)
				require.Equal(t, http.StatusOK, response.Code, response.Body.String())
				assert.Equal(t, plain.Body.Bytes(), response.Body.Bytes())
			}
			assert.Equal(t, int32(2), requests.Load(), "Base64 transports must reuse cached originals")
			pattern := filepath.Join(backend.Config.ImageCacheDir, "composition-v1-*", "*.png")
			if hashNames {
				pattern = filepath.Join(backend.Config.ImageCacheDir, "*.png")
			}
			derivatives, err := filepath.Glob(pattern)
			require.NoError(t, err)
			require.Len(t, derivatives, 1, "all transports must share one derivative")
		})
	}
}

func TestCompositionHTTPRejectsInvalidBase64BeforeFetching(t *testing.T) {
	backend, requests := compositionTestBackend(t)
	encoded := base64.StdEncoding.EncodeToString([]byte(compositionTestScene))
	for _, values := range []url.Values{
		{"putb": {""}}, {"putb": {"%%%"}}, {"putb": {"Zh=="}},
		{"putb": {encoded + "\n"}}, {"putb": {encoded + " "}},
		{"putb": {base64.StdEncoding.EncodeToString([]byte{0xff})}},
		{"putb": {base64.StdEncoding.EncodeToString([]byte("not JSON"))}},
		{"putb": {base64.StdEncoding.EncodeToString([]byte(`[{"type":"unknown"}]`))}},
		{"putb": {encoded, encoded}}, {"putb": {encoded}, "put": {compositionTestScene}},
	} {
		target := "/background.png?" + values.Encode()
		assert.Equal(t, http.StatusBadRequest, serveCompositionTest(backend, target).Code)
	}
	assert.Zero(t, requests.Load())
	target := "/background.png?putb=" + url.QueryEscape(encoded)
	backend.Config.MaxCompositionNodes = 1
	assert.Equal(t, http.StatusRequestEntityTooLarge, serveCompositionTest(backend, target).Code)
	backend.Config.Server.MaxRequestURIBytes = len(target) - 1
	assert.Equal(t, http.StatusRequestURITooLong, serveCompositionTest(backend, target).Code)
	assert.Zero(t, requests.Load())
}

func TestCompositionHTTPBase64OnlyRequiresSignature(t *testing.T) {
	backend, requests := compositionTestBackend(t)
	backend.Config.DisableHashChecking = false
	encoded := base64.RawURLEncoding.EncodeToString([]byte(compositionTestScene))
	target := "/background.png?putb=" + encoded
	assert.Equal(t, http.StatusForbidden, serveCompositionTest(backend, target).Code)
	assert.Zero(t, requests.Load())
	signed := target + "&hash=" + generateRequestHash(target, backend.Config)
	decodeCompositionTest(t, serveCompositionTest(backend, signed))
	changed := base64.RawURLEncoding.EncodeToString([]byte(strings.Replace(compositionTestScene, "25", "20", 1)))
	assert.Equal(t, http.StatusForbidden, serveCompositionTest(backend, strings.Replace(signed, encoded, changed, 1)).Code)
	assert.Equal(t, int32(2), requests.Load())
}

func TestCompositionHTTPRenderingModeSeparatesCaches(t *testing.T) {
	for _, hashNames := range []bool{false, true} {
		t.Run(map[bool]string{false: "directories", true: "hashed filenames"}[hashNames], func(t *testing.T) {
			backend, requests := compositionTestBackend(t)
			backend.Config.HashCacheFilenames = hashNames
			scene := `[{"type":"image","url":"/border.png","pos":[0,0],"dim":[100,100]}]`
			extra := url.Values{"canvas": {"white"}, "width": {"24"}, "height": {"24"}}
			smooth := serveCompositionTest(backend, compositionTarget(scene, extra))
			decodeCompositionTest(t, smooth)
			pixelScene := strings.Replace(scene, `"type":"image"`, `"type":"image","mod":"pixelated"`, 1)
			pixelated := serveCompositionTest(backend, compositionTarget(pixelScene, extra))
			img := decodeCompositionTest(t, pixelated)
			assert.NotEqual(t, smooth.Body.Bytes(), pixelated.Body.Bytes())
			assert.Equal(t, color.NRGBAModel.Convert(color.White), color.NRGBAModel.Convert(img.At(5, 9)))
			assert.Equal(t, color.NRGBA{R: 80, G: 80, B: 80, A: 255}, color.NRGBAModel.Convert(img.At(6, 9)))
			autoScene := strings.Replace(scene, `"type":"image"`, `"type":"image","mod":"auto"`, 1)
			auto := serveCompositionTest(backend, compositionTarget(autoScene, extra))
			require.Equal(t, http.StatusOK, auto.Code)
			assert.Equal(t, smooth.Body.Bytes(), auto.Body.Bytes(), "explicit auto must match the default")
			repeated := serveCompositionTest(backend, compositionTarget(pixelScene, extra))
			require.Equal(t, http.StatusOK, repeated.Code)
			assert.Equal(t, pixelated.Body.Bytes(), repeated.Body.Bytes())
			assert.Equal(t, int32(1), requests.Load(), "both modes reuse the same original")
		})
	}
}

func TestCompositionHTTPStretchSupportsBothTransportsAndSeparatesFitCaches(t *testing.T) {
	for _, mod := range []string{"auto", "pixelated"} {
		for _, hashNames := range []bool{false, true} {
			t.Run(mod+"/"+map[bool]string{false: "directories", true: "hashed filenames"}[hashNames], func(t *testing.T) {
				backend, requests := compositionTestBackend(t)
				backend.Config.HashCacheFilenames = hashNames
				outputs := make(map[string][]byte)
				for _, fit := range []string{"contain", "cover", "stretch"} {
					scene := `[{"type":"image","url":"/border.png","pos":[0,0],"dim":[100,100],"fit":"` + fit + `","mod":"` + mod + `"}]`
					extra := url.Values{"canvas": {"white"}, "width": {"24"}, "height": {"12"}, "type": {"png"}}
					plain := serveCompositionTest(backend, compositionTarget(scene, extra))
					img := decodeCompositionTest(t, plain)
					assert.Equal(t, image.Rect(0, 0, 24, 12), img.Bounds())
					outputs[fit] = plain.Body.Bytes()
					extra.Del("put")
					extra.Set("putb", base64.RawURLEncoding.EncodeToString([]byte(scene)))
					encoded := serveCompositionTest(backend, "/background.png?"+extra.Encode())
					require.Equal(t, http.StatusOK, encoded.Code, encoded.Body.String())
					assert.Equal(t, outputs[fit], encoded.Body.Bytes())
				}
				assert.NotEqual(t, outputs["contain"], outputs["stretch"])
				assert.NotEqual(t, outputs["cover"], outputs["stretch"])
				assert.NotEqual(t, outputs["contain"], outputs["cover"])
				assert.Equal(t, int32(1), requests.Load(), "all fits must reuse the same original")
				pattern := filepath.Join(backend.Config.ImageCacheDir, "composition-v1-*", "*.png")
				if hashNames {
					pattern = filepath.Join(backend.Config.ImageCacheDir, "*.png")
				}
				derivatives, err := filepath.Glob(pattern)
				require.NoError(t, err)
				require.Len(t, derivatives, 3, "different fits need separate derivatives, transports must share them")
			})
		}
	}
}

func TestCompositionHTTPKeepsCanvasSizeAndSeparatesSceneCaches(t *testing.T) {
	for _, hashNames := range []bool{false, true} {
		t.Run(map[bool]string{false: "directories", true: "hashed filenames"}[hashNames], func(t *testing.T) {
			backend, requests := compositionTestBackend(t)
			backend.Config.HashCacheFilenames = hashNames
			target := compositionTarget(compositionTestScene, nil)
			first := serveCompositionTest(backend, target)
			img := decodeCompositionTest(t, first)
			require.Equal(t, image.Rect(0, 0, 8, 8), img.Bounds())
			assert.Equal(t, color.NRGBA{R: 80, G: 80, B: 80, A: 255}, color.NRGBAModel.Convert(img.At(2, 3)))
			assert.Equal(t, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, color.NRGBAModel.Convert(img.At(2, 2)))
			second := serveCompositionTest(backend, target)
			require.Equal(t, first.Body.Bytes(), second.Body.Bytes())
			assert.Equal(t, int32(2), requests.Load(), "background and child should both come from cache")
			changed := strings.Replace(compositionTestScene, `"pos":[25,25]`, `"pos":[0,0]`, 1)
			third := serveCompositionTest(backend, compositionTarget(changed, nil))
			decodeCompositionTest(t, third)
			assert.NotEqual(t, first.Body.Bytes(), third.Body.Bytes())
			assert.Equal(t, int32(2), requests.Load(), "new layout should reuse cached originals")
		})
	}
}

func TestCompositionHTTPSharpenAppliesToFinalComposition(t *testing.T) {
	backend, _ := compositionTestBackend(t)
	before := decodeCompositionTest(t, serveCompositionTest(backend, compositionTarget(compositionTestScene, nil)))
	after := decodeCompositionTest(t, serveCompositionTest(backend, compositionTarget(compositionTestScene, url.Values{"sharpen": {".6"}})))
	expected := imaging.Sharpen(before, .6)
	changed := false
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			assert.Equal(t, color.NRGBAModel.Convert(expected.At(x, y)), color.NRGBAModel.Convert(after.At(x, y)))
			changed = changed || color.NRGBAModel.Convert(before.At(x, y)) != color.NRGBAModel.Convert(after.At(x, y))
		}
	}
	assert.True(t, changed, "sharpening must affect child edges, not just the white background")
}

func TestCompositionHTTPSignsWholeSceneWithoutChildHashes(t *testing.T) {
	backend, requests := compositionTestBackend(t)
	backend.Config.DisableHashChecking = false
	target := compositionTarget(compositionTestScene, nil)
	assert.Equal(t, http.StatusForbidden, serveCompositionTest(backend, target).Code)
	assert.Zero(t, requests.Load())
	signed := target + "&hash=" + generateRequestHash(target, backend.Config)
	decodeCompositionTest(t, serveCompositionTest(backend, signed))
	assert.Equal(t, int32(2), requests.Load())
	changed := strings.Replace(signed, "25", "20", 1)
	assert.Equal(t, http.StatusForbidden, serveCompositionTest(backend, changed).Code)
	assert.Equal(t, int32(2), requests.Load())
	// A put-only request must also be protected, without width/height/sharpen.
	unsigned := "/background.png?put=" + url.QueryEscape(compositionTestScene)
	assert.Equal(t, http.StatusForbidden, serveCompositionTest(backend, unsigned).Code)
}

func TestCompositionHTTPRejectsInvalidChildURLsBeforeFetching(t *testing.T) {
	backend, requests := compositionTestBackend(t)
	for _, childURL := range []string{
		"https://example.com/child.png", "//example.com/child.png", "child.png", "/../child.png", "/%2e%2e/child.png",
		"/child.png#fragment", "/child.png?put=[]", "/child.png?putb=W10", "/child.png?width=-1", "/child.png?width=2&width=3",
		"/child.png?sharpen=NaN", "/child.png?profile=missing", "/child.png?unexpected=1", "/child.png?type=invalid",
	} {
		t.Run(childURL, func(t *testing.T) {
			value, err := json.Marshal(childURL)
			require.NoError(t, err)
			scene := `[{"type":"image","url":` + string(value) + `,"pos":[0,0],"dim":[100,100]}]`
			assert.Equal(t, http.StatusBadRequest, serveCompositionTest(backend, compositionTarget(scene, nil)).Code)
		})
	}
	assert.Zero(t, requests.Load())
	// Encoded delimiters stay part of the path rather than turning into an
	// origin query. This path contains a literal question mark in its filename.
	scene := `[{"type":"image","url":"/has%3Fmark.png?mode=proportional","pos":[0,0],"dim":[100,null]}]`
	decodeCompositionTest(t, serveCompositionTest(backend, compositionTarget(scene, nil)))
}

func TestCompositionHTTPPropagatesChildFailuresWithoutCachingPartialResult(t *testing.T) {
	for _, tc := range []struct {
		path   string
		status int
	}{{"/missing.png", http.StatusNotFound}, {"/corrupt.png", http.StatusUnsupportedMediaType}} {
		t.Run(tc.path, func(t *testing.T) {
			backend, _ := compositionTestBackend(t)
			scene := strings.Replace(compositionTestScene, "/child.png", tc.path, 1)
			target := compositionTarget(scene, nil)
			for range 2 {
				response := serveCompositionTest(backend, target)
				assert.Equal(t, tc.status, response.Code)
				assert.Equal(t, "text/plain; charset=utf-8", response.Header().Get("Content-Type"))
				assert.NotContains(t, response.Body.String(), tc.path)
			}
		})
	}
}

func TestCompositionHTTPEnforcesTotalBudgetsAndSceneLimits(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*tools.Config)
	}{{"nodes", func(c *tools.Config) { c.MaxCompositionNodes = 1 }},
		{"depth", func(c *tools.Config) { c.MaxCompositionDepth = 1 }},
		{"decoded pixels", func(c *tools.Config) { c.MaxCompositionSourcePixels = 70 }},
		{"render pixels", func(c *tools.Config) { c.MaxCompositionRenderPixels = 100 }}} {
		t.Run(tc.name, func(t *testing.T) {
			backend, _ := compositionTestBackend(t)
			tc.change(&backend.Config)
			assert.Equal(t, http.StatusRequestEntityTooLarge, serveCompositionTest(backend, compositionTarget(compositionTestScene, nil)).Code)
		})
	}
	backend, _ := compositionTestBackend(t)
	target := compositionTarget(compositionTestScene, nil)
	decodeCompositionTest(t, serveCompositionTest(backend, target))
	backend.Config.MaxCompositionSourcePixels = 70
	assert.Equal(t, http.StatusRequestEntityTooLarge, serveCompositionTest(backend, target).Code, "tightened limits must not reuse an old cached composition")
}

func TestCompositionHTTPChildProfileOriginChangesInvalidateCache(t *testing.T) {
	backend, _ := compositionTestBackend(t)
	blue := compositionTestPNG(4, 2, color.NRGBA{B: 255, A: 255})
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(blue) }))
	defer origin.Close()
	backend.Config.Profiles = []tools.Profile{{Name: "child", OriginalBackend: backend.Config.Server.OriginalBackend}}
	scene := strings.Replace(compositionTestScene, "/child.png", "/child.png?profile=child", 1)
	target := compositionTarget(scene, nil)
	before := decodeCompositionTest(t, serveCompositionTest(backend, target))
	backend.Config.Profiles[0].OriginalBackend = origin.URL
	after := decodeCompositionTest(t, serveCompositionTest(backend, target))
	assert.NotEqual(t, color.NRGBAModel.Convert(before.At(2, 3)), color.NRGBAModel.Convert(after.At(2, 3)))
	assert.Equal(t, color.NRGBA{B: 255, A: 255}, color.NRGBAModel.Convert(after.At(2, 3)))
}

func TestCompositionHTTPWorksWithOneWorkerAndSharesConcurrentRequests(t *testing.T) {
	backend, _ := compositionTestBackend(t)
	target := compositionTarget(compositionTestScene, nil)
	var group sync.WaitGroup
	for range 8 {
		group.Go(func() {
			assert.Equal(t, http.StatusOK, serveCompositionTest(backend, target).Code)
		})
	}
	group.Wait()
}

func TestCompositionHTTPConcurrentLayoutsShareCompleteSourceFiles(t *testing.T) {
	backend, _ := compositionTestBackend(t)
	backend.processingSlots = make(chan struct{}, 4)
	var group sync.WaitGroup
	for i := range 8 {
		group.Go(func() {
			scene := strings.Replace(compositionTestScene, `"pos":[25,25]`, `"pos":[`+string(rune('0'+i))+`,0]`, 1)
			response := serveCompositionTest(backend, compositionTarget(scene, nil))
			assert.Equal(t, http.StatusOK, response.Code)
		})
	}
	group.Wait()
}

func TestCompositionHTTPChildProfileAutocropPrecedesAutoDimensionAndDoesNotInheritRoot(t *testing.T) {
	backend, _ := compositionTestBackend(t)
	backend.Config.Profiles = []tools.Profile{{Name: "child", Width: 8, Height: 8, Mode: "proportional"}}
	scene := `[{"type":"image","url":"/border.png?profile=child&autocrop=15,0","pos":[0,0],"dim":[50,null]}]`
	img := decodeCompositionTest(t, serveCompositionTest(backend, compositionTarget(scene, url.Values{"width": {"40"}, "height": {"40"}, "mode": {"stretch"}})))
	assert.Equal(t, image.Rect(0, 0, 40, 40), img.Bounds())
	assert.Equal(t, color.NRGBA{R: 80, G: 80, B: 80, A: 255}, color.NRGBAModel.Convert(img.At(19, 9)))
	assert.Equal(t, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, color.NRGBAModel.Convert(img.At(19, 10)))
}

func TestCompositionHTTPAutoDimensionIncludesChildCanvasTransformations(t *testing.T) {
	backend, _ := compositionTestBackend(t)
	scene := `[{"type":"image","url":"/child.png?width=4&height=4&mode=contain","pos":[0,0],"dim":[50,null]}]`
	extra := url.Values{"width": {"40"}, "height": {"40"}}
	contain := decodeCompositionTest(t, serveCompositionTest(backend, compositionTarget(scene, extra)))
	proportional := decodeCompositionTest(t, serveCompositionTest(backend, compositionTarget(strings.Replace(scene, "mode=contain", "mode=proportional", 1), extra)))
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	assert.NotEqual(t, white, color.NRGBAModel.Convert(contain.At(10, 12)))
	assert.Equal(t, white, color.NRGBAModel.Convert(proportional.At(10, 12)))
}

func TestCompositionHTTPThreeImageLayout(t *testing.T) {
	backend, _ := compositionTestBackend(t)
	scene := `[
 {"type":"rect","pos":[5,5],"dim":[60,90],"anc":"center center","cld":[{"type":"image","url":"/red.png","pos":[-50,-50],"dim":[100,100]}]},
 {"type":"rect","pos":[70,5],"dim":[10,90],"anc":"center center","cld":[{"type":"image","url":"/green.png","pos":[-50,-50],"dim":[100,100]}]},
 {"type":"rect","pos":[80,5],"dim":[15,90],"anc":"center center","cld":[{"type":"image","url":"/blue.png","pos":[-50,-50],"dim":[100,100]}]}
]`
	img := decodeCompositionTest(t, serveCompositionTest(backend, compositionTarget(scene, url.Values{"width": {"600"}, "height": {"600"}})))
	assert.Equal(t, image.Rect(0, 0, 600, 600), img.Bounds())
	assert.Equal(t, color.NRGBA{R: 255, A: 255}, color.NRGBAModel.Convert(img.At(30, 210)))
	assert.Equal(t, color.NRGBA{R: 255, A: 255}, color.NRGBAModel.Convert(img.At(389, 389)))
	assert.Equal(t, color.NRGBA{G: 255, A: 255}, color.NRGBAModel.Convert(img.At(450, 300)))
	assert.Equal(t, color.NRGBA{B: 255, A: 255}, color.NRGBAModel.Convert(img.At(525, 300)))
	assert.Equal(t, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, color.NRGBAModel.Convert(img.At(410, 300)))
}

func TestCompositionHTTPNegotiatesWebPAndSupportsHEAD(t *testing.T) {
	backend, _ := compositionTestBackend(t)
	target := "/background.png?put=" + url.QueryEscape(compositionTestScene)
	request := httptest.NewRequest(http.MethodHead, target, nil)
	request.Header.Set("Accept", "image/webp")
	response := httptest.NewRecorder()
	backend.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "image/webp", response.Header().Get("Content-Type"))
	assert.Equal(t, "Accept", response.Header().Get("Vary"))
	assert.NotEmpty(t, response.Header().Get("Content-Length"))
	assert.Empty(t, response.Body.Bytes())
}

func TestCompositionHTTPWhiteCanvasSkipsBackgroundAndSeparatesCache(t *testing.T) {
	for _, hashNames := range []bool{false, true} {
		t.Run(map[bool]string{false: "directories", true: "hashed filenames"}[hashNames], func(t *testing.T) {
			backend, requests := compositionTestBackend(t)
			backend.Config.HashCacheFilenames = hashNames
			backend.Config.MaxCompositionSourcePixels = 100
			target := compositionTarget(compositionTestScene, url.Values{"canvas": {"white"}, "width": {"8"}, "height": {"8"}})
			first := serveCompositionTest(backend, target)
			img := decodeCompositionTest(t, first)
			assert.Equal(t, image.Rect(0, 0, 8, 8), img.Bounds())
			assert.Equal(t, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, color.NRGBAModel.Convert(img.At(0, 0)))
			assert.Equal(t, color.NRGBA{R: 80, G: 80, B: 80, A: 255}, color.NRGBAModel.Convert(img.At(2, 3)))
			assert.Equal(t, int32(1), requests.Load(), "must fetch only the child, never the background")
			assert.Equal(t, first.Body.Bytes(), serveCompositionTest(backend, target).Body.Bytes())
			assert.Equal(t, int32(1), requests.Load())
			// Same path and scene without the canvas flag must load the background.
			ordinary := compositionTarget(compositionTestScene, url.Values{"width": {"8"}, "height": {"8"}})
			decodeCompositionTest(t, serveCompositionTest(backend, ordinary))
			assert.Equal(t, int32(2), requests.Load())
			backend.Config.MaxCompositionSourcePixels = 8 // Only the 4x2 child is decoded.
			decodeCompositionTest(t, serveCompositionTest(backend, target))
			backend.Config.MaxCompositionSourcePixels = 7
			assert.Equal(t, http.StatusRequestEntityTooLarge, serveCompositionTest(backend, target).Code)
		})
	}
}

func TestWhiteCanvasRequiresDimensionsAndSignatureAndEnforcesRenderLimit(t *testing.T) {
	backend, requests := compositionTestBackend(t)
	for _, target := range []string{
		"/canvas.jpg?canvas=white&type=png",
		"/canvas.jpg?canvas=white&width=8&type=png",
		"/canvas.jpg?canvas=white&width=8&height=0&type=png",
		"/canvas.jpg?canvas=black&width=8&height=8&type=png",
		"/canvas.jpg?canvas=white&canvas=white&width=8&height=8&type=png",
	} {
		assert.Equal(t, http.StatusBadRequest, serveCompositionTest(backend, target).Code)
	}
	assert.Zero(t, requests.Load())
	backend.Config.Profiles = []tools.Profile{{Name: "canvas", Width: 8, Height: 8}}
	backend.Config.DisableHashChecking = false
	target := "/canvas.jpg?canvas=white&profile=canvas&type=png"
	assert.Equal(t, http.StatusForbidden, serveCompositionTest(backend, target).Code)
	img := decodeCompositionTest(t, serveCompositionTest(backend, target+"&hash="+generateRequestHash(target, backend.Config)))
	assert.Equal(t, image.Rect(0, 0, 8, 8), img.Bounds())
	assert.Zero(t, requests.Load())
	backend.Config.DisableHashChecking = true
	backend.Config.MaxCompositionRenderPixels = 10
	assert.Equal(t, http.StatusRequestEntityTooLarge, serveCompositionTest(backend, "/canvas.jpg?canvas=white&width=4&height=4&type=png").Code)
}
