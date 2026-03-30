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

### [ ] 2. Thumbnail generation

Generate a small cached thumbnail for each image and store its path in the database.

**Library:** `github.com/disintegration/imaging`

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

### [ ] 3. RAW file support

Support CR2, NEF, ARW and other RAW formats by extracting their embedded JPEG preview.

**Approach:** Use a wrapper around `exiftool` or `dcraw` to extract the embedded JPEG, then treat it like any other image for thumbnail generation and EXIF parsing.

**Steps:**

1. Add `.cr2`, `.nef`, `.arw` to the extensions list in the walker
2. Write a `ExtractRAWPreview(path, outDir string) (string, error)` function in `internal/raw/`
3. Detect RAW files by extension and run preview extraction before thumbnail generation
4. Fall back gracefully if `exiftool`/`dcraw` is not installed — log a warning, skip the file

**Done when:** RAW files appear in the DB with thumbnails generated from their embedded previews.
