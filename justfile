dev:
    go run ./cmd/main.go & cd web && npm run dev

build:
    cd web && npm run build
    go build ./cmd/main.go
