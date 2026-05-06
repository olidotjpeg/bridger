# Bridger — Todo

## Done

- **Phase 1 – Foundation:** Go module, SQLite schema + migrations, directory scanner, upsert pipeline, CLI entry point
- **Phase 2 – Image Processing:** EXIF extraction, thumbnail generation, RAW preview support (CR2/NEF/ARW/RAF)
- **Phase 3 – API:** HTTP server (Gin), `GET /api/images`, `GET /api/images/:id/full`, `GET /api/tags`, `POST /api/scan`, `GET /api/scan/status`
- **Phase 4 – React UI:** Vite + TanStack Query setup, masonry grid, lightbox, keyboard navigation
- **Phase 5 – Write Operations:** Filtering/sorting, `PATCH /api/images/:id`, `POST /api/tags`, star rating UI, tag management, bulk operations
- **Phase 6 – Live Watching:** `fsnotify` watcher, recursive subdirectory watching, scan progress in sidebar

---

## Backlog

### [ ] Cull mode

A distraction-free single-image view for rapid picking. Accessible from the gallery via a toolbar toggle.

- Full-height image, no sidebar overlap
- Filmstrip of thumbnails along the bottom
- Keyboard: `←` / `→` to advance, `1`–`5` to rate, `x` to reject, `Escape` to return to grid
- Navigates the full current query result, not just the current page

---

### [ ] EXIF detail panel

Surface camera metadata in the lightbox.

- Add `camera_model`, `iso`, `aperture`, `shutter_speed`, `focal_length` columns (migration + EXIF extraction update)
- Collapsible panel below the tag editor showing: camera model, focal length, aperture, shutter speed, ISO

---

### [ ] Date grouping

Replace flat pagination with date-grouped sections.

- `GET /api/dates` returning `[{ date: "2025-04-05", count: 42 }]`
- Sticky date headers in the gallery with image counts per shoot day
- Preserve filter/sort state within each group

---

### [ ] Cull related RAW/JPEG pairs

When a shoot contains both a RAW file and a same-named JPEG (e.g. `DSC_001.NEF` + `DSC_001.JPG`), culling decisions on one should optionally propagate to the other.

**Backend:**
- On scan, detect pairs by matching base filename (sans extension) within the same directory; store a `paired_image_id` foreign key on both rows (nullable, self-referencing `images.id`)
- `PATCH /api/images/:id` accepts a `propagate_to_pair: true` flag; if set and a pair exists, apply the same rating/tags to the paired image atomically

**Frontend:**
- In the lightbox/cull mode, show a badge indicating a paired file exists (e.g. "RAW+JPG")
- Add a toggle in settings: "Always propagate culling to paired file" (default off)
- When the toggle is off, show a one-time confirmation prompt the first time a user rates a paired image

**Edge cases to handle:**
- Pair breaks if one file is deleted from disk (set `paired_image_id` to NULL on next scan)
- Many cameras write multi-shot bursts with the same base name — only pair when extensions are one RAW + one JPEG
