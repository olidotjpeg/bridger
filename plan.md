# Bridger → Wails Desktop App

## Goal

Replace the CLI HTTP server with a proper desktop app using [Wails v2](https://wails.io). Users double-click, a window opens. No browser, no ports, no terminal.

The key insight that keeps this migration small: **Wails' asset server accepts a standard `http.Handler`**. Gin implements `http.Handler`. So the entire `internal/` API layer stays completely unchanged.

---

## Prerequisites

Install the Wails CLI:
```
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Wails also needs system webview dependencies:
- **macOS**: nothing extra (uses WKWebView)
- **Linux**: `libgtk-3-dev libwebkit2gtk-4.0-dev`
- **Windows**: Edge WebView2 (ships with Windows 11/Edge; bootstrapper available for older systems)

Verify with `wails doctor`.

---

## What to Delete

The following files in `cmd/` are no longer needed — Wails handles all of this:

- `static_dev.go` / `static_prod.go` — Wails serves static assets
- `browser_dev.go` / `browser_prod.go` — Wails opens the window

---

## What to Add

### 1. `go.mod` — add the dependency

```
go get github.com/wailsapp/wails/v2
```

### 2. `cmd/embed.go` — move the embed here

Move the `//go:embed web/dist` declaration out of `static_prod.go` into a new standalone file. Keep the `!dev` build tag.

### 3. `cmd/main.go` — replace `router.Run()` with `wails.Run()`

The main function setup (config, db, migrations, scanner, router) is identical to what's there now. The only change is at the end: instead of calling `router.Run()`, you call `wails.Run()` and pass it an `*options.App`.

The critical option is `AssetServer`, which takes:
- `Assets`: your `embed.FS` (the frontend)
- `Handler`: the Gin router (handles `/api/*` and `/thumbs/*`)

Wails will serve static files from `Assets` first; anything not found there falls through to `Handler`. SPA fallback to `index.html` is automatic.

You can also set window size, title, and macOS-specific options (like `TitleBarHiddenInset` for the native translucent toolbar look).

### 4. `wails.json` — project config at the repo root

This tells the Wails CLI where the frontend lives, how to build it, and what to run in dev mode. See [wails.io/docs/reference/project-config](https://wails.io/docs/reference/project-config).

Key fields:
- `frontend.dir` → `"web"`
- `frontend.install` → `"npm install"`
- `frontend.build` → `"npm run build"`
- `frontend.dev:serverUrl` → `"auto"` (Wails injects the dev server URL)

### 5. `justfile` — replace build recipes

Replace `go run`, `npm run dev` combos with Wails CLI equivalents:

- `wails dev` — starts both Go backend and Vite, opens app window with hot-reload
- `wails build` — compiles and bundles; on macOS produces a `.app` automatically
- `wails build -platform windows/amd64` — cross-targets (CGO complicates this; use CI for cross-builds)

---

## The Scanner Startup Problem

Currently, the scanner starts before the server and the server blocks. With Wails, `wails.Run()` is the blocking call.

Start the scanner in the `OnStartup` callback (or just before `wails.Run()` as a goroutine, same as now). The `OnStartup` callback receives a `context.Context` from Wails — useful if you want to bind any Go methods to the frontend later.

---

## Optional: Native File Picker

The setup screen currently uses a plain text input for the photo directory path. With Wails you can replace it with a native OS file picker:

1. Create a small `App` struct with a `ctx context.Context`
2. Add a method that calls `runtime.OpenDirectoryDialog()`
3. Register it via `Bind` in `wails.Run()`
4. Call it from the frontend with the auto-generated JS binding

This is a noticeable UX improvement for non-technical users. Can be done as a second pass after the basic migration works.

---

## Dev Workflow After Migration

```
wails dev          # replaces: just dev
wails build        # replaces: just build
go test -tags dev ./...   # unchanged
```

---

## Migration Order (suggested)

1. Add `wailsapp/wails/v2` to `go.mod`
2. Create `wails.json`
3. Create `cmd/embed.go`, delete the four old `cmd/*.go` files
4. Rewrite `cmd/main.go` — get it compiling with `wails.Run()`
5. Run `wails dev`, verify the window opens and `/api/ping` works
6. Run `wails build`, verify the `.app` opens and the full app works
7. Update `justfile`
8. (Optional) Add native file picker for the setup flow
