# An opinionated image-server written in go

A reverse proxy that brings images to a specific size, can automatically crop away excess white space, and has a few other features.

This server was written primarily for content management systems where images uploaded by less technical users should be easily corrected and output in different sizes at the end.

Example:

```http
GET http://127.0.0.1:8080/some-image-name.jpg?width=600&height=600&autocrop=15,3&sharpen=.6&output=webp&hash=58e7e17786
```

You can set the parameters as you like. However, you still have to specify a hash value, which is generated from the request URI without hash value. This way you keep control over which image formats can end up in the cache.

It's also possible to specify profiles via the config.toml, for which you then do not need a hash value.

```
GET http://127.0.0.1:8080/some-image-name.jpg?profile=500px
```

The original images are pulled from any source. This works as long as the images of the original source are only accessible by a path and do not require any further query-parameters.