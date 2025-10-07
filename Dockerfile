FROM golang:1.24-alpine AS build

WORKDIR /app

# Copy source code
COPY . .
RUN go mod download

# Build the application from cmd directory
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o p2p-call ./cmd/main.go

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Copy the binary
COPY --from=build /app/p2p-call .

# Copy .env file only if it exists (using wildcard)
COPY --from=build /app/.env* ./

# Make binary executable
RUN chmod +x p2p-call
EXPOSE 8443

# Run the application
CMD ["./p2p-call"]
