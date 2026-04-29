dev:
    #!/bin/bash
    go run -tags dev ./cmd &
    until curl -sf http://localhost:8080/api/ping > /dev/null 2>&1; do sleep 0.5; done
    cd web && npm run dev

build:
    cd web && npm run build
    CGO_ENABLED=0 go build -o bridger ./cmd

build-windows:
    cd web && npm run build
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/bridger-windows-amd64.exe ./cmd

build-linux:
    cd web && npm run build
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/bridger-linux-amd64 ./cmd

build-mac-arm:
    cd web && npm run build
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o dist/bridger-darwin-arm64 ./cmd

build-mac-amd:
    cd web && npm run build
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o dist/bridger-darwin-amd64 ./cmd

build-all: build-windows build-linux build-mac-arm build-mac-amd

test:
    CGO_ENABLED=0 go test -tags dev ./...
