export PATH := env_var('HOME') + "/go/bin:" + env_var('PATH')

dev:
    wails dev -tags dev

build:
    wails build -o dist/Bridger

build-mac-arm:
    wails build -platform darwin/arm64

build-mac-amd:
    wails build -platform darwin/amd64

build-all: build-mac-arm build-mac-amd

test:
    CGO_ENABLED=0 go test -tags dev ./internal/...

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
