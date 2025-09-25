FROM golang:1.24-alpine AS build

WORKDIR /app

# opusfile installation dependencies
RUN apk update && apk add --no-cache \
    gcc \
    g++ \
    musl-dev \
    opus-dev \
    opusfile-dev \
    pkgconfig \
    make

ENV PKG_CONFIG_PATH="/usr/lib/pkgconfig:/usr/share/pkgconfig"
ENV CGO_ENABLED=1
ENV GOOS=linux

COPY . .
RUN go mod download

# check opus
RUN pkg-config --exists opus && echo "Opus found" || echo "Opus NOT found"
RUN pkg-config --cflags --libs opus || echo "pkg-config failed for opus"

# Build the application with verbose logging
RUN go build -v -x -ldflags="-s -w" -o p2p-call ./cmd/main.go 2>&1 | tee /tmp/build.log || \
    (echo "=== Build failed ===" && cat /tmp/build.log && exit 1)

# Runtime stage
FROM alpine:latest

WORKDIR /app

RUN apk update && \
    apk add --no-cache opus opusfile

COPY --from=build /app/p2p-call .
COPY --from=build /app/.env* ./

RUN chmod +x p2p-call
EXPOSE 8443

CMD ["./p2p-call"]