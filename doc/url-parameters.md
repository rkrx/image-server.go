# URL parameters

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

## Example

```http
GET http://127.0.0.1:8080/some-image-name.jpg?width=600&height=600&autocrop=15,3&sharpen=.6&type=webp&hash=77f018d1
```

You can set the transformation parameters as needed, but they require a hash
generated from the request URI without the hash value. The output `type` can be
changed without generating another hash so one signed transformation can be
delivered as JPEG, WebP, or AVIF. See [Request signing](signing.md) for details.

It's also possible to specify profiles via the `config.toml`, for which you do
not need a hash value.

```
GET http://127.0.0.1:8080/some-image-name.jpg?profile=500px
```

The original images are pulled from any source. This works as long as the images
from the original source are accessible by path only and do not require any
additional query parameters.
