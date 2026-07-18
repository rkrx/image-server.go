# An opinionated image server written in Go

A reverse proxy that resizes images to a specific size, can automatically crop away excess whitespace, and has a few other features. This project follows the [RESTful Image API Specification](https://github.com/riapi/riapi) with some additions and was primarily written for content management systems where images uploaded by less technical users should be easily corrected and output in different sizes as needed.

## Todo

- https://github.com/muesli/smartcrop

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

For a local non-containerized run, use Go 1.26.5 and set the same environment
variable before starting the server.

## Principle of operation

### Parameters

| Parameter type    | Syntax          | Possible value pattern                        | Description                                                                           |
|-------------------|-----------------|-----------------------------------------------|---------------------------------------------------------------------------------------|
| Width constraint  | `width` or `w`  | Positive integer                              | Set the width of the resulting image. (_Optional_)                                    |
| Height constraint | `height` or `h` | Positive integer                              | Set the height of the resulting image. (_Optional_)                                   |
| Profile           | `profile`       | <`name of the profile`>                       | Profiles can be configured separately in the config file. (_Optional_)                |
| Mode              | `mode`          | `proportional`, `contain`, `cover`, `stretch` | The operational mode when the image is resized.                                       |
| Output format     | `type`          | `jpeg`, `webp`, `avif`, `png`, `gif`          | Explicitly select the output format. (_Optional_)                                     |
| Autocrop          | `autocrop`      | `threshold`, `border`                         | The settings for autocrop.                                                            |
| Parameter-Hash    | `hash`          | HEX-Code                                      | A hex code hash to prevent unauthorized or unwanted image requests from being cached. |
| API version       | `v`             | `0.1..∞`                                      | The API version, currently 1.0 for this project.                                      |

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
