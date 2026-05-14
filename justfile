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

release bump="patch":
    #!/bin/bash
    set -e
    git fetch --tags
    latest=$(git tag --sort=-v:refname | grep '^v[0-9]' | head -1)
    if [ -z "$latest" ]; then latest="v0.0.0"; fi
    IFS='.' read -r major minor patch <<< "${latest#v}"
    case "{{bump}}" in
      major) major=$((major+1)); minor=0; patch=0 ;;
      minor) minor=$((minor+1)); patch=0 ;;
      patch) patch=$((patch+1)) ;;
    esac
    next="v${major}.${minor}.${patch}"
    echo "Current: $latest → Next: $next"
    read -p "Tag and push $next? [y/N] " confirm
    [[ "$confirm" == [yY] ]] || exit 0
    git tag "$next"
    git push origin "$next"
