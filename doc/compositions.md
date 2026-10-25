# Image compositions

The `put` parameter overlays images on an existing background image:

```text
/canvas.jpg?width=600&height=600&sharpen=.6&type=webp&put=<URL-encoded JSON>&hash=<signature>
```

Without `canvas=white`, `/canvas.jpg` must exist at the configured origin.
The background's `autocrop`, size and `mode` transformations run first;
their result becomes the fixed root canvas. Elements are then painted, the outer
`sharpen` is applied to the completed composition, and the result is encoded.
Placement never enlarges the root canvas. Existing requests without `put` or
`putb` retain their transformation order and cache keys.

Add `canvas=white` to start with a generated white canvas instead. Both width
and height must be positive, either from URL parameters or the selected profile.
The root path is only used to identify the cached result in this mode; no root
image is fetched. For example, `/canvas.jpg?canvas=white&width=542&height=318&put=...`.
Root `mode` and `autocrop` have no effect on this uniform background. Child
transformations and final sharpening work as usual. Blank canvases have separate
cache entries from origin-backed images and are subject to the output and render
limits. A generated background does not consume the source-decoding pixel budget.

For local tests, set `disableHashChecking = true` in the local `config.toml`.
Restart the server after editing this file; it reads configuration at startup.
The URLs then need no `hash`, including requests with `put`, `putb` or
`canvas=white`. An editable 542 x 318 local example using `kamera.jpg`,
`plus.png` and `objektiv.jpg` is in [collage-local.json](examples/collage-local.json),
with its ready-to-use URL in [collage-local-url.txt](examples/collage-local-url.txt).

Each element requires `type`, `pos` and `dim`. Use `type` consistently (not
`typ`). Positions and dimensions accept fractional percentages of the
corresponding parent axis: x/width use parent width; y/height use parent height.

| Field        | Meaning                                                                                                                                                                                                                                                                          |
|--------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `type`       | `rect` for a layout container, or `image` for a leaf image.                                                                                                                                                                                                                      |
| `pos: [x,y]` | The target point relative to the parent's coordinate anchor. With `piv`, places that point of the element at the target; without it, retains the existing top-left positioning before signed dimensions are normalized. Negative positions and positions above 100% are allowed. |
| `dim: [w,h]` | Signed percentages of the parent dimensions. Without `piv`, a negative width extends left and a negative height extends up. With explicit `piv`, absolute sizes are used around the selected visible pivot. Neither mirrors the image.                                           |
| `piv`        | Rectangles and images: the point of the element's own layout box placed at `pos`, for example `center center`. Not inherited. Omission preserves existing positioning.                                                                                                           |
| `anc`        | Rectangles only: coordinate origin for children. Default: `left top`. The root always uses `left top`.                                                                                                                                                                           |
| `cld`        | Rectangles only: array of children. Rectangles do not paint a background of their own.                                                                                                                                                                                           |
| `ovf`        | Rectangles only: `visible` (default) or `hidden`. Hidden overflow clips all descendants at this rectangle. Ancestor clips remain active.                                                                                                                                         |
| `url`        | Images only: an origin-relative path, optionally with image transformation parameters.                                                                                                                                                                                           |
| `fit`        | Images only: `contain` (default) shows the whole image; `cover` fills the box and clips excess image content at the image's own box; `stretch` fills the box exactly, allowing the aspect ratio to change.                                                                       |
| `alg`        | Images only: aligns the image content inside its box, or selects the crop alignment for `cover`. Default: `center center`. It does not move the box. For `stretch`, it is validated but has no visible effect.                                                                   |
| `mod`        | Images only: `auto` (default) resizes with Lanczos3; `pixelated` uses nearest-neighbor sampling to preserve hard pixel edges.                                                                                                                                                    |

`mod` controls the final resizing into the image's layout box, for `contain`,
`cover` and `stretch`. It does not change transformations in the child's URL or
final canvas sharpening and encoding. For pixel art such as the 3 x 3 `plus.png`,
set `"mod": "pixelated"` on the image element. The local example already uses it.

`anc`, `piv` and `alg` use a horizontal word followed by a vertical word,
separated by one space: `left`, `center` or `right`, then `top`, `center` or
`bottom`. For example, `right bottom` is valid. With a parent anchor of
`center center`, a child without `piv` at `pos: [-50,-50]`, `dim: [100,100]`
exactly covers the parent. A child without `piv` at `[0,0]` starts at the parent
center; its own center is not placed there.

`piv` moves the element's whole layout box, while `alg` arranges image content
inside that box. The pivot uses the final dimensions, including any `null`
dimension derived from the transformed source. With explicit `piv`, `left`,
`right`, `top` and `bottom` always refer to the visible edges, even for negative
dimensions. For example, `pos: [50,50], dim: [-20,-20], piv: "center center"`
centers a 20% x 20% box on the parent center when its `anc` is `left top`.
Without `piv`, negative dimensions retain their existing directed expansion;
therefore explicit `piv: "left top"` differs from omission for negative sizes.

For three plus signs centered horizontally at 25%, 50% and 75% of a parent
with `anc: "left top"`, use:

```json
[
  {"type":"image","url":"/plus.png","pos":[50,25],"dim":[10,null],"piv":"center center","mod":"pixelated"},
  {"type":"image","url":"/plus.png","pos":[50,50],"dim":[10,null],"piv":"center center","mod":"pixelated"},
  {"type":"image","url":"/plus.png","pos":[50,75],"dim":[10,null],"piv":"center center","mod":"pixelated"}
]
```

For `anc: "center center"`, use `[0,-25]`, `[0,0]` and `[0,25]` instead.
An editable example is in [three-plus-local.json](examples/three-plus-local.json)
with a ready-to-use [Base64url URL](examples/three-plus-local-putb-url.txt)
for a white 400 x 400 PNG canvas without sharpening. Both `put` and `putb`
support `piv`; invalid values return 400.

Negative dimensions are normalized to a rectangle before laying out its children.
For example, without `piv`, `pos: [20,0]`, `dim: [-30,100]` spans x = -10%
through +20%, and children use that normalized rectangle. Geometry is rounded at
drawing boundaries; nested percentage calculations retain fractional precision.

An image may have one `null` dimension. It is calculated from the other dimension
and the image's aspect ratio **after its URL transformations**, using the
absolute size of the specified dimension. Both dimensions cannot be `null`;
rectangles require two numeric dimensions. A zero-sized box paints nothing. With
one auto dimension, all three fit modes produce the same fit, and there is no
spare space inside the box for alignment. To center an image within a larger
container, use `dim: [100,100]`, `fit: "contain"` and `alg: "center center"`
instead.

For example, `"dim": [100,100], "fit": "stretch"` resizes the transformed source
to the full parent size without cropping; a wide image placed in a square box
is distorted to a square. This works with both `put` and `putb`. Negative
dimensions without `piv` still extend left/up without mirroring, and parent/root
clipping continues to apply.

Child URLs support their own `profile`, `width`, `height`, `mode`, `autocrop`
and `sharpen`. They do not inherit the outer transformations. Their selected
profile determines the configured origin, and they are processed internally
without HTTP calls back to this server or intermediate image encoding. A child
`type` can be supplied but does not select an intermediate encoding; the outer
`type` controls the final output. A child `hash` is unnecessary: the outer
signature authorizes the complete JSON and all child operations. External
absolute URLs, directory traversal, unknown or repeated child parameters, and
nested `put` or `putb` URLs are rejected.

Elements paint in depth-first array order; later elements cover earlier ones.
Image transparency is blended over the existing content. Rectangles default to
visible overflow, but everything is clipped at the root boundary. Unknown JSON
fields and invalid geometry are rejected. A failed child load or transformation
fails the entire request; no partial derivative is cached. Invalid specifications
return 400, invalid signatures return 403, processing limits return 413, and
child origin/decode errors use the corresponding error status.

For a camera, plus sign and lens, the decoded `put` value can be:

```json
[
  {
    "type": "rect",
    "pos": [5, 5],
    "dim": [60, 90],
    "anc": "center center",
    "cld": [{
      "type": "image",
      "url": "/kamera.jpg?autocrop=15,3&mode=proportional",
      "pos": [-50, -50],
      "dim": [100, 100],
      "fit": "contain",
      "alg": "center center"
    }]
  },
  {
    "type": "rect",
    "pos": [70, 5],
    "dim": [10, 90],
    "anc": "center center",
    "cld": [{
      "type": "image",
      "url": "/plus.png",
      "mod": "pixelated",
      "pos": [-50, -50],
      "dim": [100, 100],
      "fit": "contain",
      "alg": "center center"
    }]
  },
  {
    "type": "rect",
    "pos": [80, 5],
    "dim": [15, 90],
    "anc": "center center",
    "cld": [{
      "type": "image",
      "url": "/objektiv.jpg?autocrop=15,3&mode=proportional",
      "pos": [-50, -50],
      "dim": [100, 100],
      "fit": "contain",
      "alg": "center center"
    }]
  }
]
```

`mode=proportional` in these child URLs preserves the cropped aspect ratio
rather than adding the existing `contain` mode's background canvas before
placement. Only the outer URL needs sharpening in this example. Adding child
sharpening would apply it in addition to the final sharpening.

Encode the complete JSON as a query parameter, including the `&` characters
inside child URLs. For example, with the JSON array above stored in `elements`:

```javascript
const query = new URLSearchParams({
  width: "600", height: "600", sharpen: ".6", type: "webp",
  put: JSON.stringify(elements)
});
const requestURI = "/canvas.jpg?" + query.toString();
```

Alternatively, pass the UTF-8 JSON as Base64 in `putb`. Standard Base64 (`+` and
`/`) and URL-safe Base64 (`-` and `_`) are supported, with or without `=`
padding. Use exactly one `put` or `putb` parameter. Repeated parameters, both
formats together, malformed Base64, whitespace in Base64, or an invalid JSON
layout return 400. URL-safe Base64 is convenient for copying URLs; standard
Base64 must still be URL-encoded so `+` is not interpreted as a space. For
example:

```javascript
const jsonBytes = new TextEncoder().encode(JSON.stringify(elements));
const putb = btoa(Array.from(jsonBytes, byte => String.fromCharCode(byte)).join(""))
  .replaceAll("+", "-").replaceAll("/", "_").replace(/=+$/, "");
const query = new URLSearchParams({
  canvas: "white", width: "542", height: "318", type: "webp", putb
});
const requestURI = "/canvas.jpg?" + query.toString();
```

Both formats share layout validation, processing limits and derivative cache
entries for the same decoded composition. The request URI limit also applies
to Base64 requests; Base64 does not compress the JSON. A ready-to-use local
Base64url example is in [collage-local-putb-url.txt](examples/collage-local-putb-url.txt).

Generate the signature server-side from this encoded request URI using the
existing hash procedure, then append `&hash=...`. The derivative cache includes
the full layout, resolved child profiles and origins, output format and encoder
settings. Processing-limit changes also invalidate composition derivatives.

Compositions retain the existing per-image, request-URI and processing-worker
limits and add the following configurable limits:

```toml
maxCompositionNodes = 64
maxCompositionImages = 32
maxCompositionDepth = 8
maxCompositionSourcePixels = 80000000
maxCompositionRenderPixels = 80000000
```

Depth starts at 1 for elements directly inside the root. Source pixels count
each decode, including the background and repeated uses of the same source.
Render pixels count cumulative processing work for crop, resize, sharpening,
canvas and placement operations, including intermediate dimensions; this is a
work budget, not just the final output size. Individual generated image
dimensions must also respect the existing image limits. Derived crop dimensions
are checked before allocation. Zero or negative limit settings use the safe
defaults.
