FROM golang:1.18.0-alpine3.15
RUN apk add gcc g++ zlib zlib-dev
WORKDIR /app
COPY src /app/src
COPY go.mod /app/
COPY go.sum /app/
COPY main.go /app/
COPY config.toml /app/
RUN mkdir /var/cache/image-server
VOLUME /var/cache/image-server
EXPOSE 8080/tcp
ENV GO_IMG_SRV_VAR_CACHE_DIR=/var/cache/image-server
CMD ["go", "run", "main.go"]