FROM golang:1.24-alpine AS build

WORKDIR /app

# Добавить opusfile-dev для сборки
RUN apt-get update && apt-get -y install libopus-dev libopusfile-dev

ENV CGO_ENABLED=1
ENV GOOS=linux

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the application
RUN go build -ldflags="-s -w" -o p2p-call ./cmd/main.go

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Добавить opusfile для runtime
RUN apk add --no-cache opus opusfile

COPY --from=build /app/p2p-call .
COPY --from=build /app/.env* ./

RUN chmod +x p2p-call
EXPOSE 8443

CMD ["./p2p-call"]