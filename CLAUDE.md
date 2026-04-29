# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Project Is

Bridger is a self-hosted photo culling and management tool. It scans photo directories, extracts EXIF metadata, generates thumbnails, and provides a web UI for browsing, rating, tagging, and bulk-culling large photo libraries.

## Commands

### Development
```bash
just dev -- --dir /path/to/photos   # Run backend + Vite dev server together
just build                           # Production binary with embedded frontend
just test                            # Run all Go tests
```

### Go Backend
```bash
CGO_ENABLED=0 go run -tags dev ./cmd/main.go --dir /path/to/photos
CGO_ENABLED=0 go test -tags dev ./...                     # All tests
CGO_ENABLED=0 go test -v -tags dev ./internal/db          # Single package, verbose
CGO_ENABLED=0 go test -run TestFunctionName -tags dev ./... # Single test
```

### Frontend
```bash
cd web && npm install
npm run dev    # Vite dev server at :5173 (proxies /api and /thumbs to :8080)
npm run build  # Outputs to ../cmd/web/dist (embedded in production binary)
npm run lint
```

### Runtime Flags
- `--dir` (default `.`) — root directory to scan for photos
- `--db` (default `./bridger.db`) — SQLite database path
- `--thumbs` (default `./thumbs`) — thumbnail cache directory

## Architecture

### Overview
The backend is a single Go binary that does two things concurrently: serves an HTTP API (Gin on `:8080`) and runs background goroutines that scan, watch, and index photo files. The frontend is a React SPA embedded into the production binary via `go:embed`.

### Backend Packages (`internal/`)
- **`api/`** — Gin HTTP handlers; all routes are prefixed `/api`
- **`db/`** — SQLite CRUD and schema migrations (`golang-migrate`); three tables: `images`, `tags`, `image_tags`
- **`scanner/`** — Orchestrates directory traversal, EXIF extraction, thumbnail generation, and DB upserts; runs in background goroutine; exposes scan progress
- **`walker/`** — Filesystem traversal, filters to supported extensions (JPEG, PNG, RAW formats)
- **`watcher/`** — `fsnotify`-based watcher that triggers re-index when files are added/modified
- **`thumbs/`** — Thumbnail generation via `disintegration/imaging` (pure Go, no CGO)
- **`raw/`** — RAW preview extraction: pure Go, extracts the embedded JPEG preview from the RAW binary
- **`exif/`** — EXIF metadata extraction via `rwcarlsen/goexif`

### Frontend (`web/src/`)
- **State**: TanStack Query manages all server state (images, tags, scan status); no global client-side state store
- **Key flows**: Gallery (masonry grid) → Lightbox (full-res) → star rating or tag edit → `PATCH /api/images/:id` → TanStack Query cache invalidation
- **BulkActionBar**: activates on multi-select, sends batched PATCH requests

### Dev vs. Prod Build Split
- `cmd/static_dev.go` (build tag `dev`) — proxies static assets to Vite dev server
- `cmd/static_prod.go` (no tag) — embeds `web/dist` into binary with `go:embed`
- The `just dev` recipe and `go test -tags dev` both require the `dev` build tag

### Database Schema
Migrations live in `sql/migrations/` as numbered up/down SQL files. The `db` package applies them automatically on startup.

## Critical Dependencies

No native dependencies required. The binary is pure Go with `CGO_ENABLED=0`:
- **`modernc.org/sqlite`** — pure Go SQLite port, no C compiler needed
- **`disintegration/imaging`** — pure Go image resizing for thumbnails
- RAW previews extracted from the embedded JPEG inside each RAW file (pure Go)

Cross-compilation works out of the box: `GOOS=windows GOARCH=amd64 go build ./cmd`

## Supported Image Formats

JPEG (`.jpg`, `.jpeg`), PNG (`.png`), RAW: Canon CR2, Nikon NEF, Sony ARW, Fujifilm RAF
