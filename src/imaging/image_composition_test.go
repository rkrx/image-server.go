package imaging

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func compositionSolid(w, h int, c color.Color) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(c), image.Point{}, draw.Src)
	return img
}

func renderCompositionTest(t *testing.T, w, h int, scene string, source image.Image) image.Image {
	t.Helper()
	nodes, err := ParseComposition(scene, CompositionLimits{100, 50, 10})
	require.NoError(t, err)
	result, err := Compose(context.Background(), compositionSolid(w, h, color.White), nodes,
		func(context.Context, string) (image.Image, error) { return source, nil },
		func(int, int) error { return nil })
	require.NoError(t, err)
	return result
}

func assertCompositionColor(t *testing.T, img image.Image, x, y int, expected color.Color) {
	t.Helper()
	assert.Equal(t, color.NRGBAModel.Convert(expected), color.NRGBAModel.Convert(img.At(x, y)), "pixel %d,%d", x, y)
}

func TestCompositionPivotAllPointsAndParentAnchors(t *testing.T) {
	parent := compositionBox{x: 20, y: 30, width: 40, height: 60}
	for _, typ := range []string{"rect", "image"} {
		for ax, horizontal := range []string{"left", "center", "right"} {
			for ay, vertical := range []string{"top", "center", "bottom"} {
				for px, ownHorizontal := range []string{"left", "center", "right"} {
					for py, ownVertical := range []string{"top", "center", "bottom"} {
						t.Run(typ+"/"+horizontal+" "+vertical+"/"+ownHorizontal+" "+ownVertical, func(t *testing.T) {
							scene := fmt.Sprintf(`[{"type":"%s","pos":[10,-10],"dim":[50,50],"piv":"%s %s"%s}]`, typ, ownHorizontal, ownVertical, map[string]string{"rect": "", "image": `,"url":"/child"`}[typ])
							nodes, err := ParseComposition(scene, CompositionLimits{10, 10, 10})
							require.NoError(t, err)
							box, err := layoutCompositionBox(parent, float64(ax)/2, float64(ay)/2, nodes[0], nil)
							require.NoError(t, err)
							assert.Equal(t, compositionBox{
								x:     []float64{24, 44, 64}[ax] - []float64{0, 10, 20}[px],
								y:     []float64{24, 54, 84}[ay] - []float64{0, 15, 30}[py],
								width: 20, height: 30,
							}, box)
						})
					}
				}
			}
		}
	}
}

func TestCompositionPivotUsesAbsoluteAndAutoDimensions(t *testing.T) {
	parent := compositionBox{width: 200, height: 100}
	source := compositionSolid(4, 2, color.Black)
	for _, dim := range []string{
		`[40,40]`, `[-40,40]`, `[40,-40]`, `[-40,-40]`,
		`[40,null]`, `[-40,null]`, `[null,40]`, `[null,-40]`,
	} {
		t.Run(dim, func(t *testing.T) {
			scene := fmt.Sprintf(`[{"type":"image","url":"/child","pos":[50,50],"dim":%s,"piv":"center center"}]`, dim)
			nodes, err := ParseComposition(scene, CompositionLimits{10, 10, 10})
			require.NoError(t, err)
			box, err := layoutCompositionBox(parent, 0, 0, nodes[0], source)
			require.NoError(t, err)
			assert.Equal(t, compositionBox{x: 60, y: 30, width: 80, height: 40}, box)
		})
	}
	for _, tc := range []struct {
		piv  string
		x, y float64
	}{{"", 20, 10}, {`,"piv":"left top"`, 100, 50}, {`,"piv":"right bottom"`, 20, 10}} {
		scene := `[{"type":"rect","pos":[50,50],"dim":[-40,-40]` + tc.piv + `}]`
		nodes, err := ParseComposition(scene, CompositionLimits{10, 10, 10})
		require.NoError(t, err)
		box, err := layoutCompositionBox(parent, 0, 0, nodes[0], nil)
		require.NoError(t, err)
		assert.Equal(t, compositionBox{x: tc.x, y: tc.y, width: 80, height: 40}, box)
	}
}

func TestCompositionPivotKeepsFractionalGeometryAndLimits(t *testing.T) {
	nodes, err := ParseComposition(`[{"type":"rect","pos":[50,50],"dim":[30,40],"piv":"center center"}]`, CompositionLimits{10, 10, 10})
	require.NoError(t, err)
	box, err := layoutCompositionBox(compositionBox{width: 13, height: 7}, 0, 0, nodes[0], nil)
	require.NoError(t, err)
	assert.InDelta(t, 4.55, box.x, 1e-9)
	assert.InDelta(t, 2.1, box.y, 1e-9)
	assert.InDelta(t, 3.9, box.width, 1e-9)
	assert.InDelta(t, 2.8, box.height, 1e-9)
	_, err = layoutCompositionBox(compositionBox{width: 1e12, height: 1e12}, 0, 0, nodes[0], nil)
	require.ErrorIs(t, err, ErrCompositionLimit)
}

func TestCompositionPivotUsesLayoutBoxWhileAlignmentControlsContent(t *testing.T) {
	for _, fit := range []string{"contain", "cover", "stretch"} {
		t.Run(fit, func(t *testing.T) {
			scene := fmt.Sprintf(`[{"type":"image","url":"/child","pos":[50,50],"dim":[40,80],"piv":"center center","fit":"%s","alg":"left top"}]`, fit)
			img := renderCompositionTest(t, 100, 100, scene, compositionSolid(4, 2, color.Black))
			assertCompositionColor(t, img, 30, 10, color.Black)
			assertCompositionColor(t, img, 29, 10, color.White)
			assertCompositionColor(t, img, 70, 10, color.White)
			assertCompositionColor(t, img, 50, 90, color.White)
			if fit == "contain" {
				assertCompositionColor(t, img, 50, 29, color.Black)
				assertCompositionColor(t, img, 50, 30, color.White)
				assertCompositionColor(t, img, 50, 50, color.White)
			} else {
				assertCompositionColor(t, img, 50, 50, color.Black)
				assertCompositionColor(t, img, 69, 89, color.Black)
			}
		})
	}
}

func TestCompositionRectanglePivotMovesChildAnchorWithoutInheritance(t *testing.T) {
	for _, dim := range []string{`[40,40]`, `[-40,-40]`} {
		scene := fmt.Sprintf(`[{"type":"rect","pos":[50,50],"dim":%s,"piv":"center center","anc":"center center","ovf":"hidden","cld":[{"type":"image","url":"/child","pos":[-50,-50],"dim":[100,100],"fit":"stretch"}]}]`, dim)
		img := renderCompositionTest(t, 100, 100, scene, compositionSolid(4, 2, color.Black))
		for y := 0; y < 100; y++ {
			for x := 0; x < 100; x++ {
				var expected color.Color = color.White
				if image.Pt(x, y).In(image.Rect(30, 30, 70, 70)) {
					expected = color.Black
				}
				assertCompositionColor(t, img, x, y, expected)
			}
		}
	}
}

func TestCompositionPivotRespectsParentAndRootClipping(t *testing.T) {
	for _, overflow := range []string{"visible", "hidden"} {
		t.Run(overflow, func(t *testing.T) {
			scene := fmt.Sprintf(`[{"type":"rect","pos":[50,50],"dim":[40,40],"piv":"center center","anc":"center center","ovf":"%s","cld":[{"type":"image","url":"/child","pos":[0,0],"dim":[400,400],"piv":"center center","fit":"stretch"}]}]`, overflow)
			img := renderCompositionTest(t, 100, 100, scene, compositionSolid(4, 2, color.Black))
			assert.Equal(t, image.Rect(0, 0, 100, 100), img.Bounds())
			painted := img.Bounds()
			if overflow == "hidden" {
				painted = image.Rect(30, 30, 70, 70)
			}
			for y := 0; y < 100; y++ {
				for x := 0; x < 100; x++ {
					var expected color.Color = color.White
					if image.Pt(x, y).In(painted) {
						expected = color.Black
					}
					assertCompositionColor(t, img, x, y, expected)
				}
			}
		})
	}
}

func TestCompositionPixelatedPreservesTransparentPlusEdges(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	magenta := color.NRGBA{R: 255, B: 128, A: 255}
	for y := 0; y < 3; y++ {
		for x := 0; x < 3; x++ {
			if x == 1 || y == 1 {
				source.SetNRGBA(x, y, magenta)
			}
		}
	}
	scene := `[{"type":"image","url":"/plus","pos":[0,0],"dim":[100,100],"mod":"pixelated"}]`
	pixelated := renderCompositionTest(t, 24, 24, scene, source)
	smooth := renderCompositionTest(t, 24, 24, `[{"type":"image","url":"/plus","pos":[0,0],"dim":[100,100]}]`, source)
	for y := 0; y < 24; y++ {
		for x := 0; x < 24; x++ {
			var expected color.Color = color.White
			if x/8 == 1 || y/8 == 1 {
				expected = magenta
			}
			assertCompositionColor(t, pixelated, x, y, expected)
		}
	}
	assert.NotEqual(t, pixelated.At(7, 0), smooth.At(7, 0), "default resizing should still smooth pixel edges")
}

func TestCompositionNormalizesNegativeParentDimensionsAndUsesChildAnchor(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 6, 6))
	draw.Draw(source, source.Bounds(), image.NewUniform(color.RGBA{B: 255, A: 255}), image.Point{}, draw.Src)
	draw.Draw(source, image.Rect(0, 0, 3, 6), image.NewUniform(color.RGBA{R: 255, A: 255}), image.Point{}, draw.Src)
	img := renderCompositionTest(t, 10, 10, `[{"type":"rect","pos":[80,80],"dim":[-60,-60],"anc":"center center","cld":[{"type":"image","url":"/child","pos":[-50,-50],"dim":[100,100]}]}]`, source)
	assert.Equal(t, image.Rect(0, 0, 10, 10), img.Bounds())
	assertCompositionColor(t, img, 1, 2, color.White)
	assertCompositionColor(t, img, 2, 2, color.RGBA{R: 255, A: 255})
	assertCompositionColor(t, img, 7, 7, color.RGBA{B: 255, A: 255})
	assertCompositionColor(t, img, 8, 7, color.White)
}

func TestCompositionAllParentAnchorsPlaceTheChildTopLeft(t *testing.T) {
	for x, horizontal := range []string{"left", "center", "right"} {
		for y, vertical := range []string{"top", "center", "bottom"} {
			t.Run(horizontal+" "+vertical, func(t *testing.T) {
				scene := fmt.Sprintf(`[{"type":"rect","pos":[20,20],"dim":[60,60],"anc":"%s %s","cld":[{"type":"image","url":"/child","pos":[0,0],"dim":[20,20]}]}]`, horizontal, vertical)
				img := renderCompositionTest(t, 10, 10, scene, compositionSolid(1, 1, color.Black))
				assertCompositionColor(t, img, 2+x*3, 2+y*3, color.Black)
				assertCompositionColor(t, img, 1, 1, color.White)
			})
		}
	}
}

func TestCompositionAutoDimensionUsesSourceRatioAndParentAxis(t *testing.T) {
	for _, dim := range []string{`[50,null]`, `[null,50]`, `[-50,null]`, `[null,-50]`} {
		t.Run(dim, func(t *testing.T) {
			scene := fmt.Sprintf(`[{"type":"image","url":"/child","pos":[50,50],"dim":%s}]`, dim)
			img := renderCompositionTest(t, 20, 10, scene, compositionSolid(4, 2, color.Black))
			x, y := 10, 5
			if dim == `[-50,null]` {
				x = 0
			}
			if dim == `[null,-50]` {
				y = 0
			}
			assertCompositionColor(t, img, x, y, color.Black)
			assertCompositionColor(t, img, x+9, y+4, color.Black)
			assertCompositionColor(t, img, 0, 9, map[bool]color.Color{true: color.Black, false: color.White}[dim == `[-50,null]`])
		})
	}
}

func TestCompositionContainAlignment(t *testing.T) {
	for _, tc := range []struct {
		align string
		y     int
	}{{"center top", 0}, {"center center", 3}, {"center bottom", 5}} {
		t.Run(tc.align, func(t *testing.T) {
			scene := fmt.Sprintf(`[{"type":"image","url":"/child","pos":[0,0],"dim":[100,100],"alg":"%s"}]`, tc.align)
			img := renderCompositionTest(t, 10, 10, scene, compositionSolid(4, 2, color.Black))
			assertCompositionColor(t, img, 0, tc.y, color.Black)
			assertCompositionColor(t, img, 9, tc.y+4, color.Black)
			assertCompositionColor(t, img, 0, (tc.y+5)%10, color.White)
		})
	}
}

func TestCompositionCoverAlignmentAndImageBoxClipping(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 20, 10))
	draw.Draw(source, source.Bounds(), image.NewUniform(color.RGBA{B: 255, A: 255}), image.Point{}, draw.Src)
	draw.Draw(source, image.Rect(0, 0, 10, 10), image.NewUniform(color.RGBA{R: 255, A: 255}), image.Point{}, draw.Src)
	for _, tc := range []struct {
		align string
		color color.Color
	}{{"left center", color.RGBA{R: 255, A: 255}}, {"right center", color.RGBA{B: 255, A: 255}}} {
		t.Run(tc.align, func(t *testing.T) {
			scene := fmt.Sprintf(`[{"type":"image","url":"/child","pos":[25,0],"dim":[50,100],"fit":"cover","alg":"%s"}]`, tc.align)
			img := renderCompositionTest(t, 20, 10, scene, source)
			assertCompositionColor(t, img, 5, 0, tc.color)
			assertCompositionColor(t, img, 14, 9, tc.color)
			assertCompositionColor(t, img, 4, 5, color.White)
			assertCompositionColor(t, img, 15, 5, color.White)
		})
	}
}

func TestCompositionStretchFillsRoundedBoxWithoutCroppingOrMirroring(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	quadrants := []color.NRGBA{
		{R: 255, A: 255}, {G: 255, A: 255},
		{B: 255, A: 255}, {R: 255, G: 255, A: 255},
	}
	for y := 0; y < 2; y++ {
		for x := 0; x < 4; x++ {
			source.SetNRGBA(x, y, quadrants[y*2+x/2])
		}
	}
	for _, mod := range []string{"auto", "pixelated"} {
		for _, tc := range []struct {
			name, pos, dim string
			box            image.Rectangle
		}{
			{"positive", `[25,20]`, `[50,60]`, image.Rect(4, 4, 12, 16)},
			{"negative", `[75,80]`, `[-50,-60]`, image.Rect(4, 4, 12, 16)},
			{"fractional", `[25,20]`, `[53.125,57.5]`, image.Rect(4, 4, 13, 16)},
		} {
			t.Run(mod+"/"+tc.name, func(t *testing.T) {
				var baseline image.Image
				for _, align := range []string{"left top", "center center", "right bottom"} {
					scene := fmt.Sprintf(`[{"type":"image","url":"/child","pos":%s,"dim":%s,"fit":"stretch","mod":"%s","alg":"%s"}]`, tc.pos, tc.dim, mod, align)
					img := renderCompositionTest(t, 16, 20, scene, source)
					for y := 0; y < 20; y++ {
						for x := 0; x < 16; x++ {
							if !image.Pt(x, y).In(tc.box) {
								assertCompositionColor(t, img, x, y, color.White)
							} else {
								assert.NotEqual(t, color.NRGBAModel.Convert(color.White), color.NRGBAModel.Convert(img.At(x, y)), "box must be filled at %d,%d", x, y)
							}
						}
					}
					corners := []image.Point{
						tc.box.Min, {X: tc.box.Max.X - 1, Y: tc.box.Min.Y},
						{X: tc.box.Min.X, Y: tc.box.Max.Y - 1}, tc.box.Max.Sub(image.Pt(1, 1)),
					}
					for i, point := range corners {
						if mod == "pixelated" {
							assertCompositionColor(t, img, point.X, point.Y, quadrants[i])
						} else {
							// Lanczos can blend edge pixels slightly, but each source
							// corner must survive without being cropped or mirrored.
							actual := color.NRGBAModel.Convert(img.At(point.X, point.Y)).(color.NRGBA)
							assert.InDelta(t, quadrants[i].R, actual.R, 16)
							assert.InDelta(t, quadrants[i].G, actual.G, 16)
							assert.InDelta(t, quadrants[i].B, actual.B, 16)
							assert.Equal(t, uint8(255), actual.A)
						}
					}
					if baseline != nil {
						assert.Equal(t, baseline, img, "alignment must not affect stretch")
					}
					baseline = img
				}
			})
		}
	}
}

func TestCompositionStretchAutoDimensionKeepsSourceRatio(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	draw.Draw(source, source.Bounds(), image.NewUniform(color.Black), image.Point{}, draw.Src)
	source.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	for _, mod := range []string{"auto", "pixelated"} {
		for _, dim := range []string{`[50,null]`, `[null,50]`, `[-50,null]`, `[null,-50]`} {
			t.Run(mod+"/"+dim, func(t *testing.T) {
				scene := fmt.Sprintf(`[{"type":"image","url":"/child","pos":[50,50],"dim":%s,"fit":"stretch","mod":"%s"}]`, dim, mod)
				stretch := renderCompositionTest(t, 20, 10, scene, source)
				contain := renderCompositionTest(t, 20, 10, strings.Replace(scene, `"stretch"`, `"contain"`, 1), source)
				assert.Equal(t, contain, stretch)
			})
		}
	}
}

func TestCompositionStretchRespectsParentAndRootClipping(t *testing.T) {
	for _, mod := range []string{"auto", "pixelated"} {
		for _, overflow := range []string{"visible", "hidden"} {
			t.Run(mod+"/"+overflow, func(t *testing.T) {
				scene := fmt.Sprintf(`[{"type":"rect","pos":[25,25],"dim":[50,50],"anc":"center center","ovf":"%s","cld":[{"type":"image","url":"/child","pos":[-100,-100],"dim":[200,200],"fit":"stretch","mod":"%s"}]}]`, overflow, mod)
				img := renderCompositionTest(t, 20, 20, scene, compositionSolid(4, 2, color.Black))
				assert.Equal(t, image.Rect(0, 0, 20, 20), img.Bounds())
				painted := img.Bounds()
				if overflow == "hidden" {
					painted = image.Rect(5, 5, 15, 15)
				}
				for y := 0; y < 20; y++ {
					for x := 0; x < 20; x++ {
						var expected color.Color = color.White
						if image.Pt(x, y).In(painted) {
							expected = color.Black
						}
						assertCompositionColor(t, img, x, y, expected)
					}
				}
			})
		}
	}
}

func TestCompositionAncestorClipSurvivesVisibleDescendants(t *testing.T) {
	for _, overflow := range []string{"visible", "hidden"} {
		t.Run(overflow, func(t *testing.T) {
			scene := fmt.Sprintf(`[{"type":"rect","pos":[25,25],"dim":[50,50],"ovf":"%s","cld":[{"type":"rect","pos":[100,0],"dim":[100,100],"cld":[{"type":"image","url":"/child","pos":[-50,0],"dim":[100,100]}]}]}]`, overflow)
			img := renderCompositionTest(t, 20, 20, scene, compositionSolid(10, 10, color.Black))
			assertCompositionColor(t, img, 10, 5, color.Black)
			assertCompositionColor(t, img, 14, 14, color.Black)
			if overflow == "hidden" {
				assertCompositionColor(t, img, 15, 10, color.White)
			} else {
				assertCompositionColor(t, img, 19, 10, color.Black)
			}
			assert.Equal(t, image.Rect(0, 0, 20, 20), img.Bounds())
		})
	}
}

func TestCompositionArrayOrderBlendsAlpha(t *testing.T) {
	nodes, err := ParseComposition(`[{"type":"image","url":"/red","pos":[0,0],"dim":[100,100]},{"type":"image","url":"/blue","pos":[0,0],"dim":[100,100]}]`, CompositionLimits{10, 10, 10})
	require.NoError(t, err)
	img, err := Compose(context.Background(), compositionSolid(2, 2, color.White), nodes, func(_ context.Context, url string) (image.Image, error) {
		if url == "/red" {
			return compositionSolid(2, 2, color.NRGBA{R: 255, A: 255}), nil
		}
		return compositionSolid(2, 2, color.NRGBA{B: 255, A: 128}), nil
	}, func(int, int) error { return nil })
	require.NoError(t, err)
	assertCompositionColor(t, img, 0, 0, color.NRGBA{R: 127, B: 128, A: 255})
}

func TestCompositionRejectsInvalidSchemasAndLimits(t *testing.T) {
	for _, value := range []string{
		`null`, `{}`, `[] []`, `[null]`,
		`[{"typ":"rect","pos":[0,0],"dim":[100,100]}]`,
		`[{"type":"rect","pos":[null,0],"dim":[100,100]}]`,
		`[{"type":"rect","pos":[0,0,0],"dim":[100,100]}]`,
		`[{"type":"rect","pos":[0,0],"dim":[100,null]}]`,
		`[{"type":"rect","pos":[0,0],"dim":[100,100],"anc":"top left"}]`,
		`[{"type":"rect","pos":[0,0],"dim":[100,100],"ovf":"scroll"}]`,
		`[{"type":"rect","pos":[0,0],"dim":[100,100],"mod":"pixelated"}]`,
		`[{"type":"rect","pos":[0,0],"dim":[100,100],"piv":"top left"}]`,
		`[{"type":"image","url":"/x","pos":[0,0],"dim":[100,100],"piv":"center"}]`,
		`[{"type":"image","url":"/x","pos":[0,0],"dim":[100,100],"piv":[50,50]}]`,
		`[{"type":"image","url":"/x","pos":[0,0],"dim":[null,null]}]`,
		`[{"type":"image","url":"/x","pos":[0,0],"dim":[100,100],"cld":[]}]`,
		`[{"type":"image","url":"/x","pos":[0,0],"dim":[100,100],"fit":"invalid"}]`,
		`[{"type":"image","url":"/x","pos":[0,0],"dim":[100,100],"fit":"stretch","alg":"invalid"}]`,
		`[{"type":"image","url":"/x","pos":[0,0],"dim":[100,100],"mod":"smooth"}]`,
	} {
		t.Run(value, func(t *testing.T) {
			_, err := ParseComposition(value, CompositionLimits{10, 10, 10})
			require.Error(t, err)
		})
	}
	_, err := ParseComposition(`[{"type":"rect","pos":[0,0],"dim":[100,100],"cld":[{"type":"rect","pos":[0,0],"dim":[100,100]}]}]`, CompositionLimits{10, 10, 1})
	require.ErrorIs(t, err, ErrCompositionLimit)
	_, err = ParseComposition(`[{"type":"image","url":"/x","pos":[0,0],"dim":[100,100]},{"type":"image","url":"/y","pos":[0,0],"dim":[100,100]}]`, CompositionLimits{10, 1, 10})
	require.ErrorIs(t, err, ErrCompositionLimit)
}

func TestCompositionStopsOnSourceFailureAndPixelBudget(t *testing.T) {
	nodes, err := ParseComposition(`[{"type":"image","url":"/missing","pos":[0,0],"dim":[100,100]}]`, CompositionLimits{10, 10, 10})
	require.NoError(t, err)
	missing := errors.New("missing image")
	img, err := Compose(context.Background(), compositionSolid(10, 10, color.White), nodes,
		func(context.Context, string) (image.Image, error) { return nil, missing }, func(int, int) error { return nil })
	require.ErrorIs(t, err, missing)
	require.Nil(t, img)
	_, err = Compose(context.Background(), compositionSolid(10, 10, color.White), nodes,
		func(context.Context, string) (image.Image, error) {
			t.Fatal("must not load before budget check")
			return nil, nil
		},
		func(int, int) error { return ErrCompositionLimit })
	require.ErrorIs(t, err, ErrCompositionLimit)
}
