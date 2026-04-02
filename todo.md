# Phase 1 – Foundation

> Goal: a working CLI tool that scans a folder and saves discovered photo file paths into a SQLite database. No API, no frontend — just the plumbing.

---

## Tasks

### [X] 1. Initialise the Go module, project structure, and dependencies

Set up the Go module, create the folder layout, and pull in the SQLite driver.

**Steps:**

1. Run `go mod init github.com/yourname/bridge-clone` in the project root (replace with your actual module path).
2. Create the following folders:

```
bridge-clone/
├── cmd/
│   └── main.go          ← CLI entry point (created in task 6)
├── internal/
│   ├── db/              ← database connection and queries
│   └── indexer/         ← directory scanning logic
```

1. Add the SQLite driver — `modernc.org/sqlite` is recommended (pure Go, no C compiler needed):

```bash
go get modernc.org/sqlite
```

Then register the driver in your db package with a blank import:

```go
import _ "modernc.org/sqlite"
```

**Why `internal/`?** The `internal` directory is a Go convention that prevents other modules from importing your packages — keeps things encapsulated.

**Done when:** `go build ./...` succeeds with no errors, and the folder structure is in place.

---

### [X] 2. Write the database schema and migration runner

Create the SQLite tables and indexes the app will use. Rather than running raw SQL manually, write a simple migration runner in Go so the schema is always set up automatically on first run.

**Tables to create:**

```sql
CREATE TABLE IF NOT EXISTS images (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    file_path      TEXT NOT NULL UNIQUE,
    filename       TEXT NOT NULL,
    file_size      INTEGER,
    mime_type      TEXT,
    thumbnail_path TEXT,
    rating         INTEGER DEFAULT 0,
    capture_date   DATETIME,
    width          INTEGER,
    height         INTEGER,
    index_at     DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS tags (
    id   INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS image_tags (
    image_id INTEGER NOT NULL REFERENCES images(id),
    tag_id   INTEGER NOT NULL REFERENCES tags(id),
    PRIMARY KEY (image_id, tag_id)
);
```

> **Note:** The architecture doc also mentions a `file_hash` column for tracking files across moves/renames. This is intentionally deferred to Phase 2 when we add EXIF parsing and proper file fingerprinting. Leave it out for now.

**Indexes to create** (these make filtered queries fast):

```sql
CREATE INDEX IF NOT EXISTS idx_images_rating       ON images(rating);
CREATE INDEX IF NOT EXISTS idx_images_capture_date ON images(capture_date);
CREATE INDEX IF NOT EXISTS idx_image_tags_tag_id   ON image_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_image_tags_image_id ON image_tags(image_id);
```

**Tip:** Put all of this in an `internal/db/migrate.go` file with a single exported `RunMigrations(db *sql.DB) error` function. Call it once at startup.

**How to verify:** Write a small `main.go` that opens a DB and calls `RunMigrations`, then check with `sqlite3 bridge-clone.db '.tables'` — you should see `images`, `tags`, and `image_tags`. Run it a second time to confirm it's a no-op (no errors, no duplicates).

**Done when:** Running the migration twice produces the correct tables on first run and no errors on the second.

---

### [X] 3. Implement the directory scanner

Write a function that walks a root directory and returns a list of all photo files found.

**Use `filepath.WalkDir`** — it's the modern, efficient way to traverse directories in Go.

```go
filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
    // skip directories, check file extension, collect paths
})
```

**File extensions to support initially:** `.jpg`, `.jpeg`, `.png`
**Stretch goal:** `.cr2`, `.nef`, `.arw` (RAW formats — you can add proper processing later in Phase 2)

**Important — case sensitivity:** Cameras often write extensions in uppercase (`.JPG`, `.JPEG`). Normalise the extension to lowercase before comparing:

```go
ext := strings.ToLower(filepath.Ext(path))
```

**Error handling:** `WalkDir` will encounter permission-denied errors, symlinks to missing targets, and other OS-level issues. Don't let these crash the program. Log the error, skip the file, and keep going:

```go
if err != nil {
    log.Printf("skipping %s: %v", path, err)
    return nil // continue walking
}
```

Keep this function pure — it should just return a list of paths, not touch the database. That separation makes it easy to test.

**How to verify:** Create a test folder with 3 `.jpg` files, 1 `.png`, and 1 `.txt`. The scanner should return exactly 4 paths.

**Done when:** The function returns all photo files from a directory tree, handles uppercase extensions, and doesn't crash on unreadable files.

---

### [X] 4. Upsert discovered images into the database

For each file found by the scanner, check whether it already exists in the database. If not, insert it. If the file has changed (different size), update it. If it's unchanged, skip it.

**Identity check:** Query the database by `file_path`. If a record exists and the stored `file_size` matches the current size on disk, skip it — nothing has changed.

```go
var existingSize int64
err := db.QueryRow("SELECT file_size FROM images WHERE file_path = ?", path).Scan(&existingSize)
if err == nil && existingSize == currentFileSize {
    return // already indexed, skip
}
```

> **Why not use a full MD5/SHA1 hash?** Hashing a 30–50MB RAW file takes real time. Size + path is fast and good enough for the common case of "file hasn't changed". A full hash can be added later in Phase 2.

**Inserting / updating:** Use `INSERT ... ON CONFLICT` (not `INSERT OR REPLACE`). This is important because `INSERT OR REPLACE` actually deletes the old row first, which would wipe out any ratings or tags added later in Phase 5.

```sql
INSERT INTO images (file_path, filename, file_size, mime_type, index_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(file_path) DO UPDATE SET
    file_size  = excluded.file_size,
    index_at = excluded.index_at
```

This only updates the columns the scanner "owns" and leaves user data (rating, tags) untouched.

**Columns to populate at this stage** (EXIF parsing comes in Phase 2):

| Column | Source |
| --- | --- |
| `file_path` | Full absolute path from the scanner |
| `filename` | `filepath.Base(path)` |
| `file_size` | `os.Stat(path).Size()` |
| `mime_type` | Derive from file extension (e.g., `.jpg` → `image/jpeg`) |
| `index_at` | `time.Now().UTC()` |

**How to verify:** Run the upsert on a folder with 5 images. Check the DB has 5 rows. Run it again — all 5 should be skipped. Change one file's size (e.g., re-save it), run again — 1 updated, 4 skipped.

**Done when:** New files are inserted, unchanged files are skipped, changed files are updated without losing other columns, and the function doesn't crash on empty directories.

---

### [X] 5. Wire up the CLI entry point

Bring everything together in `cmd/main.go`. The program should accept command-line flags and run the full scan.

**CLI flags:**

```go
dir := flag.String("dir", ".", "Root directory to scan")
db  := flag.String("db", "./bridge-clone.db", "Path to the SQLite database file")
flag.Parse()
```

Using the `flag` package gives you `--help` for free.

**Minimal flow:**

1. Parse the `--dir` and `--db` flags
2. Open (or create) the SQLite database file at the `--db` path
3. Run migrations
4. Run the directory scanner on `--dir`
5. For each discovered file, run the identity check and upsert if needed
6. Print a summary to stdout

**Example output:**

```
Scanning /photos/2024...
Found 1,842 files
  Inserted: 312
  Updated:    3
  Skipped: 1,527
  Errors:     0
Done.
```

**How to verify:**

1. Run against a folder with photos — check the DB has the expected rows
2. Run again on the same folder — should show 0 inserted, all skipped
3. Run with `--help` — should print flag descriptions
4. Run with a non-existent directory — should print a clear error, not a panic

**Done when:** The CLI scans a directory, persists results to SQLite, prints a summary, and handles re-runs and bad input gracefully.

---

# Phase 2 – Image Processing

> Goal: extract metadata from photo files, generate thumbnails, and add RAW file support. Builds on Phase 1's scanner and database.

---

## Tasks

### [X] 1. EXIF extraction

Parse EXIF metadata from photo files and store it in the database.

**Library:** `github.com/rwcarlsen/goexif/exif`

**Fields to extract and store:**

| Field | DB Column |
| --- | --- |
| Capture date | `capture_date` |
| Camera model | (future — not in schema yet) |
| Image width | `width` |
| Image height | `height` |

**Steps:**

1. Add `go-exif` as a dependency
2. Write an `ExtractEXIF(path string)` function in a new `internal/exif/` package
3. Call it during the upsert loop in `main.go` and populate `capture_date`, `width`, `height`
4. Update the `ON CONFLICT DO UPDATE` in `UpsertImagePath` to also update these columns

**Error handling:** EXIF data is often missing or malformed — log and skip gracefully, don't fail the whole scan.

**Done when:** After a scan, `capture_date`, `width`, and `height` are populated for files that have valid EXIF data.

---

### [X] 2. Thumbnail generation

Generate a small cached thumbnail for each image and store its path in the database.

**Library:** `github.com/davidbyttow/govips/v2` (requires `brew install vips`)

**Spec:**
- Resize to 400px wide, maintaining aspect ratio
- Output format: JPEG or WebP
- Cache location: `~/.bridger/thumbs/` (or configurable via `--thumbs` flag)
- Filename: derived from a hash of the original path to avoid collisions

**Steps:**

1. Add `github.com/davidbyttow/govips/v2` as a dependency (`brew install vips` required)
2. Write a `GenerateThumbnail(srcPath, thumbDir string) (string, error)` function in a new `internal/thumbs/` package
3. Skip thumbnail generation if one already exists and the source file hasn't changed
4. Store the thumbnail path in `thumbnail_path` on the images row

**Done when:** After a scan, each image has a thumbnail on disk and `thumbnail_path` is populated in the DB.

---

### [X] 3. RAW file support

Support CR2, NEF, ARW and other RAW formats by extracting their embedded JPEG preview.

**Approach:** Use a wrapper around `exiftool` or `dcraw` to extract the embedded JPEG, then treat it like any other image for thumbnail generation and EXIF parsing.

**Steps:**

1. Add `.cr2`, `.nef`, `.arw` to the extensions list in the walker
2. Write a `ExtractRAWPreview(path, outDir string) (string, error)` function in `internal/raw/`
3. Detect RAW files by extension and run preview extraction before thumbnail generation
4. Fall back gracefully if `exiftool`/`dcraw` is not installed — log a warning, skip the file

**Done when:** RAW files appear in the DB with thumbnails generated from their embedded previews.

---

# Phase 3 – API (Read-only)

> Goal: expose the indexed photo library over HTTP so a frontend can consume it. All endpoints are read-only — write operations (ratings, tags) come in Phase 5.

---

## Tasks

### [X] 1. Set up the HTTP server

Add a lightweight HTTP server to the existing Go project using Go's standard library router (Go 1.22+) or Gin.

**Steps:**

1. Add the router dependency if using Gin: `go get github.com/gin-gonic/gin`
2. Create `internal/api/server.go` with a `NewServer(db *sql.DB) http.Handler` function
3. Register all routes in one place
4. Wire the server into `cmd/main.go` — it should start after migrations run

**Done when:** `go run ./cmd/main.go` starts an HTTP server and responds to requests.

---

### [X] 2. `GET /api/images`

List images with pagination and filtering.

**Query params:**

| Param | Type | Description |
| --- | --- | --- |
| `page` | int | Page number (default: 1) |
| `limit` | int | Results per page (default: 50) |
| `rating` | int | Filter by minimum rating |
| `sort` | string | `capture_date`, `rating`, `filename` (default: `capture_date`) |
| `order` | string | `asc` or `desc` (default: `desc`) |

**Response shape:**
```json
{
  "data": [
    {
      "id": 1,
      "filename": "DSC_001.jpg",
      "capture_date": "2024-12-01T10:00:00Z",
      "width": 6000,
      "height": 4000,
      "rating": 0,
      "mime_type": "image/jpeg",
      "thumbnail_path": "/api/images/1/thumbnail"
    }
  ],
  "total": 1842,
  "page": 1,
  "limit": 50
}
```

**Done when:** Endpoint returns paginated, filterable image records from the database.

---

### [X] 3. `GET /api/images/:id/thumbnail`

Serve the cached thumbnail file from disk.

**Steps:**

1. Look up `thumbnail_path` in the DB by `id`
2. Serve the file with the correct `Content-Type` header (`image/jpeg`)
3. Return `404` if the image or thumbnail doesn't exist

**Done when:** The browser can load a thumbnail by hitting this endpoint.

---

### [X] 4. `GET /api/images/:id/full`

Serve the full-resolution image from its original path on disk.

**Steps:**

1. Look up `file_path` in the DB by `id`
2. Serve the file with the correct `Content-Type` from `mime_type`
3. Return `404` if the record or file doesn't exist

**Done when:** The browser can load the full-res image by hitting this endpoint.

---

### [X] 5. `GET /api/tags`

List all tags in the database.

**Response shape:**
```json
[
  { "id": 1, "name": "Wedding" },
  { "id": 2, "name": "Landscape" }
]
```

**Done when:** Endpoint returns all tags from the `tags` table.

---

### [X] 6. `POST /api/scan` and `GET /api/scan/status`

Trigger a re-index and expose scan progress.

**`POST /api/scan`** — starts a background scan of the configured directory. Returns immediately with `202 Accepted`.

**`GET /api/scan/status`** — returns the current scan state:
```json
{
  "running": true,
  "total": 1842,
  "processed": 412,
  "errors": 3
}
```

**Steps:**

1. Move the scan loop from `main.go` into a function that can be called both at startup and via the API
2. Store scan state in a struct protected by a `sync.Mutex`
3. Expose the state via `GET /api/scan/status`

**Done when:** A POST to `/api/scan` triggers a background scan and `/api/scan/status` reflects live progress.

---

# Phase 4 – React UI (Read-only)

> Goal: a working frontend that displays the photo library using the Phase 3 API. No write operations yet — just browsing, filtering, and viewing.

---

## Tasks

### [ ] 1. Set up the React project

Scaffold the frontend inside the repo, configure the dev proxy, and wire up Go's `embed` so the production build is served from a single binary.

**Steps:**

1. Create the React app: `npm create vite@latest web -- --template react-ts`
2. Install dependencies: `npm install @tanstack/react-query zustand`
3. Configure the Vite dev proxy in `vite.config.ts` to forward `/api` and `/thumbs` to `http://localhost:8080`:
```ts
server: {
    proxy: {
        '/api': 'http://localhost:8080',
        '/thumbs': 'http://localhost:8080',
    }
}
```
4. Set the Vite build output to `web/dist`
5. Add the embed to `cmd/main.go`:
```go
//go:embed web/dist
var staticFiles embed.FS

router.StaticFS("/", http.FS(staticFiles))
```
6. Add a `justfile` with two recipes:
```just
dev:
    go run ./cmd/main.go & cd web && npm run dev

build:
    cd web && npm run build
    go build ./cmd/main.go
```

**Development:** run `just dev` — Vite serves the frontend on port 5173 and proxies API calls to Go on port 8080.

**Production:** run `just build` — produces a single Go binary with the frontend embedded.

**Done when:** `just dev` runs both servers, API requests reach Go, and `just build` produces a working single binary.

---

### [ ] 2. Image grid with virtualization

Display thumbnails in a masonry/grid layout. Must handle 10k+ images without performance issues.

**Steps:**

1. Install `react-virtuoso` or `react-window` for virtualized rendering
2. Fetch the first page of images from `GET /api/images` using TanStack Query
3. Render thumbnails using `GET /api/images/:id/thumbnail`
4. Implement infinite scroll or pagination to load more images

**Done when:** The grid renders thumbnails and loads more as the user scrolls, without rendering off-screen items.

---

### [ ] 3. Sidebar with filters

Add a sidebar for filtering and sorting the image grid.

**Controls to include:**

- Sort by: `capture_date`, `rating`, `filename`
- Order: ascending / descending
- Filter by minimum star rating (1–5 + unrated)
- Tag checkboxes (fetched from `GET /api/tags`)

**Done when:** Changing filters updates the grid in real time via TanStack Query cache invalidation.

---

### [ ] 4. Detail modal / view

Show a larger image preview with full metadata when an image is clicked.

**Content to display:**

- Full-res image via `GET /api/images/:id/full`
- Filename, capture date, dimensions, MIME type
- Star rating (display only at this stage)
- Tags applied to the image (display only)

**Done when:** Clicking an image opens the detail view with metadata and the full-res image loads.

---

### [ ] 5. Keyboard navigation

Implement keyboard shortcuts for browsing. These are core to the culling workflow.

| Key | Action |
| --- | --- |
| `←` / `→` | Previous / next image |
| `Enter` | Open detail view |
| `Escape` | Close detail view |

**Done when:** The user can navigate the library entirely by keyboard without touching the mouse.

---

# Phase 5 – Write Operations

> Goal: add rating, tagging, and bulk operations so the app becomes a usable culling tool.

---

## Tasks

### [ ] 1. `PATCH /api/images/:id` — update rating and tags

Add the write endpoint to the Go API.

**Request body:**
```json
{ "rating": 4, "tags": [1, 3] }
```

**Steps:**

1. Update `rating` on the `images` row
2. Sync `image_tags` — delete removed tags, insert new ones
3. Return the updated image record

**Done when:** A PATCH request updates rating and tags atomically without losing other fields.

---

### [ ] 2. `POST /api/tags` — create a new tag

**Request body:**
```json
{ "name": "Wedding" }
```

Returns the created tag with its `id`. Returns `409 Conflict` if the name already exists.

**Done when:** New tags can be created via the API.

---

### [ ] 3. Star rating UI

Add interactive star rating to the detail view and grid.

**Steps:**

1. Render 5 clickable stars on the detail view
2. On click, call `PATCH /api/images/:id` with the new rating
3. Invalidate the image query so the grid reflects the change immediately
4. Wire up keyboard shortcuts `1`–`5` to set rating, `0` to clear

**Done when:** Clicking stars or pressing `1`–`5` updates the rating and the grid reflects it.

---

### [ ] 4. Tag management UI

Allow adding and removing tags from images in the detail view.

**Steps:**

1. Show existing tags as removable chips
2. Add a text input with autocomplete from `GET /api/tags`
3. Allow creating new tags inline
4. On change, call `PATCH /api/images/:id` with the updated tag list

**Done when:** Tags can be added, removed, and created from the detail view.

---

### [ ] 5. Bulk operations

Allow selecting multiple images and applying a rating or tag to all of them.

**Steps:**

1. Toggle selection with `Space` or `Shift`+click
2. Show a bulk action bar when images are selected
3. Apply rating or tags to all selected images via `PATCH /api/images/:id` (one request per image)
4. Add `x` key to mark selected images as rejected (rating = -1 or a dedicated rejected flag)

**Done when:** The user can select multiple images and bulk-rate or bulk-tag them in one action.

---

# Phase 6 – Live Watching

> Goal: automatically detect new photos when they are copied from an SD card or added to the watch folder, without requiring a manual scan.

---

## Tasks

### [ ] 1. `fsnotify` folder watcher

Watch the configured photo directory for new files and trigger indexing automatically.

**Steps:**

1. Add `github.com/fsnotify/fsnotify` as a dependency
2. Create `internal/watcher/watcher.go` with a `Watch(dir string, onFile func(path string)) error` function
3. On `Create` events, check the file extension and run the existing scan pipeline (EXIF + thumbnail + upsert)
4. Start the watcher as a goroutine in `main.go` alongside the HTTP server

**Done when:** Copying a photo into the watch folder triggers automatic indexing within a few seconds.

---

### [ ] 2. Scan progress in the sidebar

Show live scan progress in the React sidebar using the `GET /api/scan/status` endpoint.

**Steps:**

1. Poll `GET /api/scan/status` every 2 seconds when `running: true` using TanStack Query
2. Display a progress bar or counter: "Scanning… 412 / 1842"
3. Stop polling when `running: false`
4. Add a manual "Scan" button that calls `POST /api/scan`

**Done when:** The sidebar shows live progress during a scan and updates automatically when new files are detected.
