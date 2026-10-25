# Getting started

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
