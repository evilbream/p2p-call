# p2p-call

## Build App

### For Windows 64-bit
```bash
GOOS=windows GOARCH=amd64 go build -o p2p-call.exe ./cmd/main.go
```

### For Windows 32-bit
```bash
GOOS=windows GOARCH=386 go build -o p2p-call-32.exe ./cmd/main.go
```

### For Windows ARM64
```bash
GOOS=windows GOARCH=arm64 go build -o p2p-call-arm64.exe ./cmd/main.go
```

## Docker

### Build Docker image
```bash
docker build -t p2p-call .
```

### Run interactively with environment file
```bash
docker run -it --env-file .env p2p-call
# HTTP port 8080 and HTTPS port 8443
docker run -it --env-file .env -p 8442:8443 p2p-call

```