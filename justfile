dev:
    #!/bin/bash
    go run ./cmd/main.go &
    until curl -sf http://localhost:8080/api/ping > /dev/null 2>&1; do sleep 0.5; done
    cd web && npm run dev

build:
    cd web && npm run build
    go build ./cmd/main.go
