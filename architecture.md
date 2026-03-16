# Architecture

## 1. High-Level Architecture

**Frontend (React):** A lightweight single-page application (SPA). Communicates with the backend via a RESTful API and displays images, ratings, and tags. Uses TanStack Query for server state (caching, pagination, background refetch, and cache invalidation — critical since the indexer mutates data in the background).

**Backend (Golang):** Serves a dual purpose:

- **API Server:** Handles requests from the React app (fetch images, update ratings, add tags, trigger scans).
- **Background Worker:** Scans designated photo directories, extracts metadata (EXIF), generates thumbnails, and updates the database.

**Database:** SQLite. Since this is an internal tool running locally or on a private server, SQLite is fast, requires zero setup, and keeps the app self-contained.

**Storage:** Local file system or an attached NAS where master photos live. Thumbnails are stored in a separate cache directory (e.g., `~/.bridge-clone/thumbs/`), referenced by path in the database.

---

## 2. Database Schema (SQLite)

| Table | Columns | Description |
| --- | --- | --- |
| `images` | `id, file_path, filename, file_hash, rating, capture_date, width, height, file_size, mime_type, thumbnail_path, indexed_at` | Core metadata. `file_hash` tracks files across moves/renames. `thumbnail_path` decouples the indexer from the API. `indexed_at` detects stale records. |
| `tags` | `id, name` | Tag library (e.g., "Wedding", "Landscape", "Client_Smith"). |
| `image_tags` | `image_id, tag_id` | Junction table for the many-to-many relationship. |

**Required indexes:**

- `images(rating)`
- `images(capture_date)`
- `image_tags(tag_id)`
- `image_tags(image_id)`

**Note on file hashing:** MD5/SHA1 on full 30–50MB RAW files is slow. Use `size + mtime + first N bytes` as a fast identity check, falling back to a full hash only on collision. For most cases, `file_path + mtime` is sufficient.

---

## 3. Backend Plan (Golang)

### A. The Indexer / Discovery Engine

- **Directory Scanning:** Use `filepath.WalkDir` to scan the root photo folder.
- **Concurrency:** Use a worker pool of 4–8 goroutines to process files in parallel without saturating the OS.
- **Job Status:** Maintain an in-memory scan job state (total files, processed, errors) so the API can expose scan progress to the UI.
- **File Watching (optional):** Use `fsnotify` to watch folders and index new files automatically when photos are copied in from an SD card.

### B. Image Processing & Metadata

- **EXIF Parsing:** Use `github.com/dsoprea/go-exif` to extract capture date, camera model, lens, and ISO.
- **Thumbnail Generation:** Generate WebP or JPEG thumbnails (e.g., 400px wide) and cache them to disk. Never send full-res files to the grid view. Use `github.com/disintegration/imaging`.
- **RAW Files:** For CR2, NEF, ARW, etc., extract the embedded JPEG preview using a wrapper around `exiftool` or `dcraw`. Treat this as a first-class concern, not an afterthought.

### C. The API Server

Use a lightweight framework: Gin, Fiber, or Go's standard library router (Go 1.22+).

#### Images

- `GET /api/images` — list images with query params: `page`, `limit`, `rating`, `tags`, `sort` (capture_date, rating, filename), `order` (asc, desc)
- `GET /api/images/:id/thumbnail` — serve cached thumbnail
- `GET /api/images/:id/full` — serve full-res image
- `PATCH /api/images/:id` — update rating and/or tags in a single request

#### Tags

- `GET /api/tags` — list all available tags (required for the sidebar)
- `POST /api/tags` — create a new tag

#### Indexer

- `POST /api/scan` — trigger a re-index of the configured directories
- `GET /api/scan/status` — poll current scan progress (total, processed, errors)

---

## 4. Frontend Plan (React)

Keep the UI clean and dark-themed (standard for photo apps — makes colors pop).

### State Management

- **TanStack Query** for all server state: image lists, tag lists, scan status. Handles caching, background refetch, and cache invalidation automatically.
- **Zustand** (or React Context) for local UI state only: selected image, active filters, multi-selection set.

### Key Components

- **Sidebar:** Tag checkboxes, star rating filter (1–5 + unrated), sort controls, search bar, and a "Scan" button with progress indicator.
- **Grid View:** Virtualized masonry layout using `react-virtuoso` or `react-window`. Only renders thumbnails visible in the viewport — essential for libraries of 10k+ photos.
- **Detail Modal / View:** Larger image preview, full EXIF data, tag input, and star rating control.

### Keyboard Shortcuts (Core UX)

Keyboard-driven culling is the primary workflow. These are non-negotiable:

| Key | Action |
| --- | --- |
| `←` / `→` | Navigate to previous/next image |
| `1` – `5` | Set star rating |
| `0` | Clear rating |
| `x` | Mark as rejected |
| `Space` | Select/deselect image (bulk ops) |
| `Enter` | Open detail view |
| `Escape` | Close detail view |

### Bulk Operations

- Select multiple images (Space or click + Shift-click)
- Bulk rate selected images
- Bulk tag selected images

---

## 5. Project Phases

| Phase | Focus |
| --- | --- |
| **1 – Foundation** | Go project structure, SQLite schema + migrations, directory scanner CLI (no API yet) |
| **2 – Image Processing** | EXIF extraction, thumbnail generation, RAW preview support |
| **3 – API (Read-only)** | HTTP server serving image list, thumbnail, full-res, tag list, scan status |
| **4 – React UI (Read-only)** | Grid view, virtualization, sidebar filters, detail modal — all read-only |
| **5 – Write Operations** | Rating + tagging via UI, keyboard shortcuts, bulk operations |
| **6 – Live Watching** | `fsnotify` integration, scan progress endpoint, sidebar scan button |
