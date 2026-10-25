# Request signing

## How to generate the hash value

The signing secret is used to generate the hash value for each request. Set it
through `GO_IMG_SRV_SECRET`; the environment value overrides a `secret` value
from `config.toml`. Keeping `secret` in the configuration file remains supported
for backwards compatibility.

The hash value is generated from the request URI without the hash value itself.
For example, if you want to request an image with the following parameters:

`/some-image-name.jpg?width=600&height=600&autocrop=15,3&sharpen=.6&type=webp`

Let's say your secret is `DontUseThisSecret123`, then you concatenate the request
URI with the secret:

`/some-image-name.jpg?width=600&height=600&autocrop=15,3&sharpen=.6&type=webp` + `DontUseThisSecret123`

By default, the server generates an `MD5` hash of this concatenated string and
takes the first eight characters of the resulting hex string. This keeps existing
URLs compatible.

To generate the final request URI, you append the hash value as a query
parameter:

`/some-image-name.jpg?width=600&height=600&autocrop=15,3&sharpen=.6&type=webp&hash=77f018d1`

## Hash configuration

```toml
hashAlgorithm = "sha256"
hashLength = 32
```

```shell
export GO_IMG_SRV_SECRET="replace-with-a-long-random-secret"
```

`hashAlgorithm` defaults to `md5` for compatibility. New installations should use
`sha256`, which is implemented as HMAC-SHA256 over the request URI without the
`hash` parameter. `hashLength` defaults to `8` and truncates the hex hash to
keep URLs short. Invalid algorithms and hash lengths outside the supported
digest size are rejected during startup.
