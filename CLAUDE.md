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
go run -tags dev ./cmd/main.go --dir /path/to/photos
go test -tags dev ./...                     # All tests
go test -v -tags dev ./internal/db          # Single package, verbose
go test -run TestFunctionName -tags dev ./... # Single test
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
- **`thumbs/`** — Thumbnail generation via `govips` (wraps libvips C library)
- **`raw/`** — RAW preview extraction: tries `govips`/libraw first, falls back to `exiftool` CLI
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

- **libvips** — required at build and runtime (`brew install vips` on macOS); govips wraps it via CGO
- **C compiler** — required by `go-sqlite3` (Xcode Command Line Tools on macOS)
- **exiftool** — optional, used as fallback for RAW preview extraction
- `CGO_ENABLED=1` is required for both sqlite3 and vips bindings

## Supported Image Formats

JPEG (`.jpg`, `.jpeg`), PNG (`.png`), RAW: Canon CR2, Nikon NEF, Sony ARW, Fujifilm RAF
