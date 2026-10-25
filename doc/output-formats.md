# Output formats and encoder defaults

## Output format negotiation

The `type` parameter explicitly selects the output format. For example,
`type=jpeg`, `type=webp`, and `type=avif` request exactly that format.

If `type` is omitted, the server negotiates the format from the `Accept` request
header. AVIF is preferred when supported, followed by WebP. JPEG is used as the
fallback. Negotiated responses include `Vary: Accept` so that HTTP caches keep
the formats separate.

## Encoder defaults

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

The cache key includes a schema version, the selected output format, and all
encoder settings. Changing an encoder setting therefore creates a new cached
derivative without reusing an image encoded with the previous settings.
