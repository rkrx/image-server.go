# An opinionated image server written in Go

A reverse proxy that resizes images to a specific size, can automatically crop away excess whitespace, and has a few other features. This project follows the [RESTful Image API Specification](https://github.com/riapi/riapi) with some additions and was primarily written for content management systems where images uploaded by less technical users should be easily corrected and output in different sizes as needed.

## Compile and run from the command line

Use Go 1.26.5, the toolchain version specified in `go.mod`. Image encoders use
CGO, so a C/C++ compiler and zlib development headers are required when building.
On macOS, install the Xcode Command Line Tools with `xcode-select --install` if
they are not already installed. On Debian/Ubuntu, install the build dependencies:

```shell
sudo apt-get update
sudo apt-get install build-essential zlib1g-dev
```

Run the following commands from the repository root. On first setup, create
`config.toml` from the sample; this command keeps an existing configuration:

```shell
test -f config.toml || cp config.sample.toml config.toml
```

Edit `config.toml` and set `[server].originalBackend` to the base URL of your
original images. Set `[server].listenAddr` to the desired address and port;
`127.0.0.1:8080` listens locally, while the sample's `0.0.0.0:8080` listens on
all interfaces. The cache directories must be writable by the user running
the server. When starting without a reverse proxy, set `[server].remoteAddrHeader`
to `""` so client-IP checks use the TCP connection instead of a proxy header.

Set your signing secret in the same shell before starting the server:

```shell
export GO_IMG_SRV_SECRET="replace-with-your-signing-secret"
```

Use your existing secret when one is already configured. This environment
variable overrides `secret` in `config.toml`; either source must provide a
non-empty secret, even when hash checking is disabled for local tests.

Compile the executable, then start it in the foreground:

```shell
CGO_ENABLED=1 go build -o image-server .
./image-server
```

Go downloads missing module dependencies during the build. With port 8080
configured, the server is reachable at `http://127.0.0.1:8080`. Keep the terminal
open while it runs; press `Ctrl+C` to stop it. Restart after changing the
configuration. Set `enableLogging = true` to see application logs on stdout.

The executable reads `config.toml` from its **current working directory**, not
from the executable's directory. Start it from the repository root, or from
another directory containing your configuration. Once compiled, Go is not
needed to run the executable; native runtime libraries must still be available.
On Debian/Ubuntu these include `libstdc++6` and `zlib1g`, as in the Docker runtime.

For development, you can also compile and start directly from the source tree:

```shell
CGO_ENABLED=1 go run .
```

This uses the same configuration and secret, and requires the build dependencies
on every machine where it is used.

## Docker

Create a local configuration and replace the example values:

```shell
cp config.sample.toml config.toml
```

Keep the signing secret outside `config.toml`. Create an environment file that
is not committed or included in the Docker build context:

```dotenv
GO_IMG_SRV_SECRET=replace-with-a-long-random-secret
```

Build the image:

```
docker build -t go-image-server .
```

Run the server with the configuration mounted read-only and a named cache
volume. The runtime container uses UID/GID `10001` and does not contain the Go
toolchain or source tree:

```shell
docker run -d --restart unless-stopped \
  --name go-image-server \
  --env-file .env \
  --mount type=bind,src="$(pwd)/config.toml",dst=/app/config.toml,readonly \
  --mount source=go-image-server-cache,dst=/var/cache/image-server \
  -p 127.0.0.1:10080:8080 \
  go-image-server
```

For a local run without Docker, see
[Compile and run from the command line](#compile-and-run-from-the-command-line).

## Principle of operation

### Parameters

| Parameter type     | Syntax          | Possible value pattern                        | Description                                                                                                               |
|--------------------|-----------------|-----------------------------------------------|---------------------------------------------------------------------------------------------------------------------------|
| Width constraint   | `width` or `w`  | Positive integer                              | Set the width of the resulting image. (_Optional_)                                                                        |
| Height constraint  | `height` or `h` | Positive integer                              | Set the height of the resulting image. (_Optional_)                                                                       |
| Profile            | `profile`       | <`name of the profile`>                       | Profiles can be configured separately in the config file. (_Optional_)                                                    |
| Mode               | `mode`          | `proportional`, `contain`, `cover`, `stretch` | The operational mode when the image is resized.                                                                           |
| Output format      | `type`          | `jpeg`, `webp`, `avif`, `png`, `gif`          | Explicitly select the output format. (_Optional_)                                                                         |
| Autocrop           | `autocrop`      | `threshold`, `border`                         | The settings for autocrop.                                                                                                |
| Sharpen            | `sharpen`       | Number from 0 to 10                           | Sharpen the image; with `put`, sharpen the completed composition.                                                         |
| Composition        | `put`           | URL-encoded JSON array                        | Place images and nested layout rectangles on the transformed background.                                                  |
| Base64 composition | `putb`          | Base64-encoded UTF-8 JSON array               | Alternative to `put`; supports standard Base64 and Base64url, with or without padding.                                    |
| Blank canvas       | `canvas`        | `white`                                       | Generate a white background instead of fetching the root image. Requires width and height, directly or through a profile. |
| Parameter-Hash     | `hash`          | HEX-Code                                      | A hex code hash to prevent unauthorized or unwanted image requests from being cached.                                     |
| API version        | `v`             | `0.1..∞`                                      | The API version, currently 1.0 for this project.                                                                          |

### How to generate the hash value

The signing secret is used to generate the hash value for each request. Set it
through `GO_IMG_SRV_SECRET`; the environment value overrides a `secret` value
from `config.toml`. Keeping `secret` in the configuration file remains supported
for backwards compatibility.

The hash value is generated from the request URI without the hash value itself. For example, if you want to request an image with the following parameters:

`/some-image-name.jpg?width=600&height=600&autocrop=15,3&sharpen=.6&type=webp`

Let's say your secret is `DontUseThisSecret123`, then you concatenate the request URI with the secret:

`/some-image-name.jpg?width=600&height=600&autocrop=15,3&sharpen=.6&type=webp` + `DontUseThisSecret123`

By default, the server generates an `MD5` hash of this concatenated string and takes the first eight characters of the resulting hex string. This keeps existing URLs compatible.

To generate the final request URI, you append the hash value as a query parameter:

`/some-image-name.jpg?width=600&height=600&autocrop=15,3&sharpen=.6&type=webp&hash=77f018d1`

### Example:

```http
GET http://127.0.0.1:8080/some-image-name.jpg?width=600&height=600&autocrop=15,3&sharpen=.6&type=webp&hash=77f018d1
```

You can set the transformation parameters as needed, but they require a hash
generated from the request URI without the hash value. The output `type` can be
changed without generating another hash so one signed transformation can be
delivered as JPEG, WebP, or AVIF.

It's also possible to specify profiles via the `config.toml`, for which you do not need a hash value.

```
GET http://127.0.0.1:8080/some-image-name.jpg?profile=500px
```

The original images are pulled from any source. This works as long as the images from the original source are accessible by path only and do not require any additional query parameters.

### Image compositions

The `put` parameter overlays images on an existing background image:

```text
/canvas.jpg?width=600&height=600&sharpen=.6&type=webp&put=<URL-encoded JSON>&hash=<signature>
```

Without `canvas=white`, `/canvas.jpg` must exist at the configured origin.
The background's `autocrop`, size and `mode` transformations run first;
their result becomes the fixed root canvas. Elements are then painted, the outer
`sharpen` is applied to the completed composition, and the result is encoded.
Placement never enlarges the root canvas. Existing requests without `put` or `putb` retain
their transformation order and cache keys.

Add `canvas=white` to start with a generated white canvas instead. Both width and
height must be positive, either from URL parameters or the selected profile.
The root path is only used to identify the cached result in this mode; no root
image is fetched. For example, `/canvas.jpg?canvas=white&width=542&height=318&put=...`.
Root `mode` and `autocrop` have no effect on this uniform background. Child
transformations and final sharpening work as usual. Blank canvases have separate
cache entries from origin-backed images and are subject to the output and render
limits. A generated background does not consume the source-decoding pixel budget.

For local tests, set `disableHashChecking = true` in the local `config.toml`.
Restart the server after editing this file; it reads configuration at startup.
The URLs then need no `hash`, including requests with `put`, `putb` or `canvas=white`.
An editable 542 x 318 local example using `kamera.jpg`, `plus.png` and
`objektiv.jpg` is in [collage-local.json](doc/examples/collage-local.json), with
its ready-to-use URL in [collage-local-url.txt](doc/examples/collage-local-url.txt).

Each element requires `type`, `pos` and `dim`. Use `type` consistently (not `typ`).
Positions and dimensions accept fractional percentages of the corresponding
parent axis: x/width use parent width; y/height use parent height.

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
`cover` and `stretch`. It does not change transformations in the child's URL or final
canvas sharpening and encoding. For pixel art such as the 3 x 3 `plus.png`, set
`"mod": "pixelated"` on the image element. The local example already uses it.

`anc`, `piv` and `alg` use a horizontal word followed by a vertical word, separated
by one space: `left`, `center` or `right`, then `top`, `center` or `bottom`.
For example, `right bottom` is valid. With a parent anchor of `center center`, a
child without `piv` at `pos: [-50,-50]`, `dim: [100,100]` exactly covers the parent. A child without `piv` at
`[0,0]` starts at the parent center; its own center is not placed there.

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
An editable example is in [three-plus-local.json](doc/examples/three-plus-local.json)
with a ready-to-use [Base64url URL](doc/examples/three-plus-local-putb-url.txt)
for a white 400 x 400 PNG canvas without sharpening. Both `put` and `putb`
support `piv`; invalid values return 400.

Negative dimensions are normalized to a rectangle before laying out its children.
For example, without `piv`, `pos: [20,0]`, `dim: [-30,100]` spans x = -10% through +20%, and
children use that normalized rectangle. Geometry is rounded at drawing boundaries;
nested percentage calculations retain fractional precision.

An image may have one `null` dimension. It is calculated from the other dimension
and the image's aspect ratio **after its URL transformations**, using the absolute
size of the specified dimension. Both dimensions cannot be `null`; rectangles
require two numeric dimensions. A zero-sized box paints nothing. With one auto
dimension, all three fit modes produce the same fit, and there is no spare space
inside the box for alignment. To center an image within a larger container, use
`dim: [100,100]`, `fit: "contain"` and `alg: "center center"` instead.

For example, `"dim": [100,100], "fit": "stretch"` resizes the transformed source
to the full parent size without cropping; a wide image placed in a square box
is distorted to a square. This works with both `put` and `putb`. Negative
dimensions without `piv` still extend left/up without mirroring, and parent/root
clipping continues to apply.

Child URLs support their own `profile`, `width`, `height`, `mode`, `autocrop` and
`sharpen`. They do not inherit the outer transformations. Their selected profile
determines the configured origin, and they are processed internally without HTTP
calls back to this server or intermediate image encoding. A child `type` can be
supplied but does not select an intermediate encoding; the outer `type` controls
the final output. A child `hash` is unnecessary: the outer signature authorizes
the complete JSON and all child operations. External absolute URLs, directory
traversal, unknown or repeated child parameters, and nested `put` or `putb` URLs are rejected.

Elements paint in depth-first array order; later elements cover earlier ones.
Image transparency is blended over the existing content. Rectangles default to
visible overflow, but everything is clipped at the root boundary. Unknown JSON
fields and invalid geometry are rejected. A failed child load or transformation
fails the entire request; no partial derivative is cached. Invalid specifications
return 400, invalid signatures return 403, processing limits return 413, and child
origin/decode errors use the corresponding error status.

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

`mode=proportional` in these child URLs preserves the cropped aspect ratio rather
than adding the existing `contain` mode's background canvas before placement.
Only the outer URL needs sharpening in this example. Adding child sharpening
would apply it in addition to the final sharpening.

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
`/`) and URL-safe Base64 (`-` and `_`) are supported, with or without `=` padding.
Use exactly one `put` or `putb` parameter. Repeated parameters, both formats
together, malformed Base64, whitespace in Base64, or an invalid JSON layout
return 400. URL-safe Base64 is convenient for copying URLs; standard Base64 must
still be URL-encoded so `+` is not interpreted as a space. For example:

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
Base64url example is in [collage-local-putb-url.txt](doc/examples/collage-local-putb-url.txt).

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

Depth starts at 1 for elements directly inside the root. Source pixels count each
decode, including the background and repeated uses of the same source. Render
pixels count cumulative processing work for crop, resize, sharpening, canvas and
placement operations, including intermediate dimensions; this is a work budget,
not just the final output size. Individual generated image dimensions must also
respect the existing image limits. Derived crop dimensions are checked before
allocation. Zero or negative limit settings use the safe defaults.

### Output format negotiation

The `type` parameter explicitly selects the output format. For example, `type=jpeg`, `type=webp`, and `type=avif` request exactly that format.

If `type` is omitted, the server negotiates the format from the `Accept` request header. AVIF is preferred when supported, followed by WebP. JPEG is used as the fallback. Negotiated responses include `Vary: Accept` so that HTTP caches keep the formats separate.

### Encoder defaults

```toml
[jpegDefaults]
quality = 85

[webpDefaults]
quality = 80
lossless = false

[avifDefaults]
quality = 60
alphaQuality = 80
speed = 6
```

The cache key includes a schema version, the selected output format, and all encoder settings. Changing an encoder setting therefore creates a new cached derivative without reusing an image encoded with the previous settings.

### Hash configuration

```toml
hashAlgorithm = "sha256"
hashLength = 32
```

```shell
export GO_IMG_SRV_SECRET="replace-with-a-long-random-secret"
```

`hashAlgorithm` defaults to `md5` for compatibility. New installations should use `sha256`, which is implemented as HMAC-SHA256 over the request URI without the `hash` parameter. `hashLength` defaults to `8` and truncates the hex hash to keep URLs short. Invalid algorithms and hash lengths outside the supported digest size are rejected during startup.

### Origin fetch limits

```toml
originFetchTimeoutInSeconds = 15
originMaxImageBytes = 52428800
```

`originFetchTimeoutInSeconds` limits how long the server waits for an original image response. `originMaxImageBytes` limits how many bytes are read from the origin before image decoding starts.

### Image processing limits

The server validates the dimensions declared by an origin image before fully
decoding it and validates requested and generated output dimensions. The safe
defaults can be adjusted for installations that intentionally process larger
assets:

```toml
maxSourceImageWidth = 12000
maxSourceImageHeight = 12000
maxSourceImagePixels = 40000000

maxOutputImageWidth = 8192
maxOutputImageHeight = 8192
maxOutputImagePixels = 20000000
```

URL dimensions must be non-negative and within these output limits. `sharpen`
accepts values from 0 to 10; autocrop tolerance accepts 0 to 100 and its border
accepts 0 to 25.

Image work is globally bounded and concurrent requests for the same uncached
derivative share one operation:

```toml
[server]
imageProcessingWorkers = 2
```

### HTTP limits

Only `GET` and `HEAD` requests are accepted. HTTP responses use
`X-Content-Type-Options: nosniff`, and error responses never include the supplied
parameter value.

```toml
[server]
readHeaderTimeoutInSeconds = 5
readTimeoutInSeconds = 15
writeTimeoutInSeconds = 120
idleTimeoutInSeconds = 60
maxHeaderBytes = 65536
maxRequestURIBytes = 8192
maxQueryParameters = 16
maxConcurrentHTTP2Streams = 100
```
