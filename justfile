dev:
    #!/bin/bash
    go run -tags dev ./cmd &
    until curl -sf http://localhost:8080/api/ping > /dev/null 2>&1; do sleep 0.5; done
    cd web && npm run dev

build:
    cd web && npm run build
    go build -o bridger ./cmd

test:
    go test -tags dev ./...
