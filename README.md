# An opinionated image server written in Go

A reverse proxy that resizes images to a specific size, can automatically crop away excess whitespace, and has a few other features. This project follows the [RESTful Image API Specification](https://github.com/riapi/riapi) with some additions and was primarily written for content management systems where images uploaded by less technical users should be easily corrected and output in different sizes as needed.

## Todo

- https://github.com/muesli/smartcrop

## Dockercod

Build the image:

```
docker build -t go-image-server .
```

Run the server:

```
docker run -d --restart unless-stopped --name go-image-server -v /var/cache/image-server:/var/cache/image-server -p 127.0.0.1:10080:8080 go-image-server
```

## Principle of operation

### Parameters

| Parameter type    | Syntax          | Possible value pattern                        | Description                                                                           |
|-------------------|-----------------|-----------------------------------------------|---------------------------------------------------------------------------------------|
| Width constraint  | `width` or `w`  | Positive integer                              | Set the width of the resulting image. (_Optional_)                                    |
| Height constraint | `height` or `h` | Positive integer                              | Set the height of the resulting image. (_Optional_)                                   |
| Profile           | `profile`       | <`name of the profile`>                       | Profiles can be configured separately in the config file. (_Optional_)                |
| Mode              | `mode`          | `proportional`, `contain`, `cover`, `stretch` | The operational mode when the image is resized.                                       |
| Output format     | `type`          | `jpeg`, `webp`, `avif`, `png`, `gif`          | Explicitly select the output format. (_Optional_)                                      |
| Autocrop          | `autocrop`      | `threshold`, `border`                         | The settings for autocrop.                                                            |
| Parameter-Hash    | `hash`          | HEX-Code                                      | A hex code hash to prevent unauthorized or unwanted image requests from being cached. |
| API version       | `v`             | `0.1..∞`                                      | The API version, currently 1.0 for this project.                                      |

### How to generate the hash value

First you have to specify the `secret`-Parameter in the `config.toml` file. This secret is used to generate the hash value for each request. You can use any string you like.

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

You can set the parameters as you like. However, you must specify a hash value, which is generated from the request URI without the hash value. This way, you retain control over which image formats can end up in the cache.

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
secret = "DontUseThisSecret123"
hashAlgorithm = "sha256"
hashLength = 16
```

`hashAlgorithm` defaults to `md5` for compatibility. New installations should use `sha256`, which is implemented as HMAC-SHA256 over the request URI without the `hash` parameter. `hashLength` defaults to `8` and truncates the hex hash to keep URLs short.

### Origin fetch limits

```toml
originFetchTimeoutInSeconds = 15
originMaxImageBytes = 52428800
```

`originFetchTimeoutInSeconds` limits how long the server waits for an original image response. `originMaxImageBytes` limits how many bytes are read from the origin before image decoding starts.
