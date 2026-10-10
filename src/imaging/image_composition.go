package imaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"io"
	"math"
	"strings"
)

var ErrCompositionLimit = errors.New("composition limit exceeded")

// CompositionNode describes a layout box. Pivot selects the point placed at Pos;
// an omitted Pivot preserves the signed top-left positioning. Anc affects children.
type CompositionNode struct {
	Type     string             `json:"type"`
	Pos      []*float64         `json:"pos"`
	Dim      []*float64         `json:"dim"`
	Pivot    string             `json:"piv,omitempty"`
	Anc      string             `json:"anc,omitempty"`
	Children []*CompositionNode `json:"cld,omitempty"`
	URL      string             `json:"url,omitempty"`
	Fit      string             `json:"fit,omitempty"`
	Align    string             `json:"alg,omitempty"`
	Mod      string             `json:"mod,omitempty"`
	Overflow string             `json:"ovf,omitempty"`
}

type CompositionLimits struct {
	MaxNodes, MaxImages, MaxDepth int
}

func ParseComposition(value string, limits CompositionLimits) ([]*CompositionNode, error) {
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.DisallowUnknownFields()
	var nodes []*CompositionNode
	if err := decoder.Decode(&nodes); err != nil || nodes == nil {
		return nil, errors.New("put must be a JSON array of layout elements")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, errors.New("unexpected data after put")
	}
	count, images := 0, 0
	var validate func([]*CompositionNode, int) error
	validate = func(elements []*CompositionNode, depth int) error {
		for _, node := range elements {
			count++
			if count > limits.MaxNodes || depth > limits.MaxDepth {
				return ErrCompositionLimit
			}
			if node == nil || len(node.Pos) != 2 || len(node.Dim) != 2 {
				return errors.New("each element requires type, pos and dim")
			}
			for _, value := range node.Pos {
				if value == nil || !finite(*value) {
					return errors.New("positions must be finite")
				}
			}
			for _, value := range node.Dim {
				if value != nil && !finite(*value) {
					return errors.New("dimensions must be finite")
				}
			}
			if node.Pivot != "" {
				if _, _, err := compositionAlignment(node.Pivot); err != nil {
					return err
				}
			}
			switch node.Type {
			case "rect":
				if node.Dim[0] == nil || node.Dim[1] == nil || node.URL != "" || node.Fit != "" || node.Align != "" || node.Mod != "" {
					return errors.New("rect requires two dimensions and cannot have image attributes")
				}
				if node.Anc == "" {
					node.Anc = "left top"
				}
				if _, _, err := compositionAlignment(node.Anc); err != nil {
					return err
				}
				if node.Overflow == "" {
					node.Overflow = "visible"
				}
				if node.Overflow != "visible" && node.Overflow != "hidden" {
					return errors.New("ovf must be visible or hidden")
				}
				if err := validate(node.Children, depth+1); err != nil {
					return err
				}
			case "image":
				images++
				if images > limits.MaxImages {
					return ErrCompositionLimit
				}
				if node.URL == "" || node.Children != nil || node.Anc != "" || node.Overflow != "" || (node.Dim[0] == nil && node.Dim[1] == nil) {
					return errors.New("image requires a URL and at least one dimension and cannot have children, anc or ovf")
				}
				if node.Fit == "" {
					node.Fit = "contain"
				}
				if node.Fit != "contain" && node.Fit != "cover" && node.Fit != "stretch" {
					return errors.New("fit must be contain, cover or stretch")
				}
				if node.Mod == "" {
					node.Mod = "auto"
				}
				if node.Mod != "auto" && node.Mod != "pixelated" {
					return errors.New("mod must be auto or pixelated")
				}
				if node.Align == "" {
					node.Align = "center center"
				}
				if _, _, err := compositionAlignment(node.Align); err != nil {
					return err
				}
			default:
				return errors.New("type must be rect or image")
			}
		}
		return nil
	}
	if err := validate(nodes, 1); err != nil {
		return nil, err
	}
	return nodes, nil
}

func compositionAlignment(value string) (float64, float64, error) {
	parts := strings.Split(value, " ")
	if len(parts) != 2 {
		return 0, 0, errors.New("alignment requires horizontal and vertical gravity")
	}
	x, xOK := map[string]float64{"left": 0, "center": .5, "right": 1}[parts[0]]
	y, yOK := map[string]float64{"top": 0, "center": .5, "bottom": 1}[parts[1]]
	if !xOK || !yOK {
		return 0, 0, errors.New("invalid alignment")
	}
	return x, y, nil
}

type compositionBox struct{ x, y, width, height float64 }

func (b compositionBox) pixels() image.Rectangle {
	return image.Rect(int(math.Round(b.x)), int(math.Round(b.y)), int(math.Round(b.x+b.width)), int(math.Round(b.y+b.height)))
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func layoutCompositionBox(parent compositionBox, ax, ay float64, node *CompositionNode, source image.Image) (compositionBox, error) {
	x := parent.x + (ax+*node.Pos[0]/100)*parent.width
	y := parent.y + (ay+*node.Pos[1]/100)*parent.height
	w, h := 0.0, 0.0
	if node.Dim[0] != nil {
		w = *node.Dim[0] / 100 * parent.width
	}
	if node.Dim[1] != nil {
		h = *node.Dim[1] / 100 * parent.height
	}
	if source != nil {
		if node.Dim[0] == nil {
			w = math.Abs(h) * float64(source.Bounds().Dx()) / float64(source.Bounds().Dy())
		}
		if node.Dim[1] == nil {
			h = math.Abs(w) * float64(source.Bounds().Dy()) / float64(source.Bounds().Dx())
		}
	}
	if node.Pivot != "" {
		px, py, err := compositionAlignment(node.Pivot)
		if err != nil {
			return compositionBox{}, err
		}
		// Explicit pivots refer to the visible edges, regardless of dimension
		// signs. Leaving piv unset retains the historical directed expansion.
		w, h = math.Abs(w), math.Abs(h)
		x -= px * w
		y -= py * h
	}
	// Bound arithmetic before conversion to integer pixel coordinates. This is
	// independent of percentages: ordinary overflows beyond 100% remain valid.
	for _, value := range []float64{x, y, w, h, x + w, y + h} {
		if !finite(value) || math.Abs(value) > 1<<30 {
			return compositionBox{}, ErrCompositionLimit
		}
	}
	if w < 0 {
		x, w = x+w, -w
	}
	if h < 0 {
		y, h = y+h, -h
	}
	return compositionBox{x, y, w, h}, nil
}

// Compose paints in depth-first array order. Clip boundaries accumulate down
// the tree, while layout keeps subpixel precision until the drawing operation.
func Compose(ctx context.Context, background image.Image, nodes []*CompositionNode,
	load func(context.Context, string) (image.Image, error), charge func(int, int) error,
) (image.Image, error) {
	bounds := background.Bounds()
	if err := charge(bounds.Dx(), bounds.Dy()); err != nil {
		return nil, err
	}
	canvas := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(canvas, canvas.Bounds(), background, bounds.Min, draw.Src)
	var paint func([]*CompositionNode, compositionBox, float64, float64, image.Rectangle) error
	paint = func(elements []*CompositionNode, parent compositionBox, ax, ay float64, clip image.Rectangle) error {
		for _, node := range elements {
			if err := ctx.Err(); err != nil {
				return err
			}
			var source image.Image
			if node.Type == "image" {
				var err error
				source, err = load(ctx, node.URL)
				if err != nil {
					return err
				}
			}
			box, err := layoutCompositionBox(parent, ax, ay, node, source)
			if err != nil {
				return err
			}
			pixels := box.pixels()
			if node.Type == "rect" {
				childClip := clip
				if node.Overflow == "hidden" {
					childClip = clip.Intersect(pixels)
				}
				childAX, childAY, _ := compositionAlignment(node.Anc)
				if err := paint(node.Children, box, childAX, childAY, childClip); err != nil {
					return err
				}
				continue
			}
			visible := clip.Intersect(pixels)
			if visible.Empty() {
				continue
			}
			if err := charge(pixels.Dx(), pixels.Dy()); err != nil {
				return err
			}
			fitWidth, fitHeight := float64(pixels.Dx()), float64(pixels.Dy())
			if node.Fit != "stretch" {
				scaleX := fitWidth / float64(source.Bounds().Dx())
				scaleY := fitHeight / float64(source.Bounds().Dy())
				scale := math.Min(scaleX, scaleY)
				if node.Fit == "cover" {
					scale = math.Max(scaleX, scaleY)
				}
				fitWidth = math.Round(float64(source.Bounds().Dx()) * scale)
				fitHeight = math.Round(float64(source.Bounds().Dy()) * scale)
			}
			if !finite(fitWidth) || !finite(fitHeight) || fitWidth > 1<<30 || fitHeight > 1<<30 {
				return ErrCompositionLimit
			}
			w := max(1, int(fitWidth))
			h := max(1, int(fitHeight))
			if err := charge(w, h); err != nil {
				return err
			}
			var img image.Image
			if node.Mod == "pixelated" {
				img = ResizePixelated(source, w, h)
			} else {
				img = Resize(source, w, h)
			}
			alignX, alignY, _ := compositionAlignment(node.Align)
			point := image.Pt(pixels.Min.X+int(math.Round(float64(pixels.Dx()-w)*alignX)), pixels.Min.Y+int(math.Round(float64(pixels.Dy()-h)*alignY)))
			destination := image.Rectangle{Min: point, Max: point.Add(image.Pt(w, h))}.Intersect(visible)
			if !destination.Empty() {
				draw.Draw(canvas, destination, img, img.Bounds().Min.Add(destination.Min.Sub(point)), draw.Over)
			}
		}
		return nil
	}
	if err := paint(nodes, compositionBox{width: float64(bounds.Dx()), height: float64(bounds.Dy())}, 0, 0, canvas.Bounds()); err != nil {
		return nil, fmt.Errorf("compose: %w", err)
	}
	return canvas, nil
}
