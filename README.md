# An image-server written in go

A reverse proxy that brings images to a specific size, can automatically crop away excess white space, and has a few other features.

Example:

```http
GET http://127.0.0.1:8080/some-image-name.jpg?width=600&height=600&autocrop=15,3&sharpen=.6&output=webp
```

The original images are pulled from any source. This works as long as the images of the original source are only accessible by a path and do not require any further query-parameters. 