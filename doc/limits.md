# Limits and quotas

## Origin fetch limits

```toml
originFetchTimeoutInSeconds = 15
originMaxImageBytes = 52428800
```

`originFetchTimeoutInSeconds` limits how long the server waits for an original
image response. `originMaxImageBytes` limits how many bytes are read from the
origin before image decoding starts.

## Image processing limits

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

## HTTP limits

Only `GET` and `HEAD` requests are accepted. HTTP responses use
`X-Content-Type-Options: nosniff`, and error responses never include the
supplied parameter value.

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
