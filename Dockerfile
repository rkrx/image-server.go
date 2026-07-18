# syntax=docker/dockerfile:1

FROM golang:1.26.5-bookworm AS builder

RUN apt-get update \
    && apt-get install -y --no-install-recommends gcc g++ zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY src ./src

RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/image-server ./main.go

FROM debian:bookworm-slim AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates libstdc++6 zlib1g \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 10001 image-server \
    && useradd --uid 10001 --gid 10001 --no-create-home --home-dir /nonexistent --shell /usr/sbin/nologin image-server \
    && mkdir -p /app /var/cache/image-server \
    && chown -R 10001:10001 /app /var/cache/image-server

WORKDIR /app

COPY --from=builder --chown=10001:10001 /out/image-server /usr/local/bin/image-server

VOLUME /var/cache/image-server
EXPOSE 8080/tcp

ENV GO_IMG_SRV_VAR_CACHE_DIR=/var/cache/image-server

USER 10001:10001

ENTRYPOINT ["/usr/local/bin/image-server"]
