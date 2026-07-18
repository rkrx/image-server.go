FROM golang:1.24-bookworm
RUN apt-get update && apt-get install -y --no-install-recommends gcc g++ zlib1g-dev && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY src /app/src
COPY go.mod /app/
COPY go.sum /app/
COPY main.go /app/
COPY config.toml /app/
RUN go get
RUN mkdir /var/cache/image-server
VOLUME /var/cache/image-server
EXPOSE 8080/tcp
ENV GO_IMG_SRV_VAR_CACHE_DIR=/var/cache/image-server
CMD ["go", "run", "main.go"]
