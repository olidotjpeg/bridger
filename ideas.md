# Ideas & Future Features

A backlog of feature ideas beyond the current roadmap.

---

## Culling Workflow

**Comparison view**
Show 2–4 images side by side to pick the best shot from a burst. Classic culling feature found in Lightroom and Capture One.

**Dedicated cull mode**
Single-image view designed for fast picking — larger image, minimal UI. Keyboard-driven: `←/→` to advance, `1–5` to rate, `x` to reject. No distractions.

**Reject cleanup**
After marking rejects (rating = -1), a "Move rejects to trash" action that either deletes or moves them to a configurable folder.

---

## Data & Discovery

**EXIF detail panel**
Surface camera model, ISO, aperture, shutter speed, and lens in the lightbox detail view. Data is already partially in the DB — just needs exposing.

**Date grouping**
Instead of flat pagination, group the grid by day or month with sticky date headers. Much easier to navigate a multi-day shoot.

**Duplicate detection**
Use perceptual hashing (`github.com/corona10/goimagehash`) to surface near-identical shots and help prune bursts quickly.

**Full-text search**
Search bar above the grid for filename, date range, or tag name. Backed by a SQLite `LIKE` or FTS5 query.

---

## Collections & Export

**Named collections**
Save a set of filters or hand-picked images as a named album. Requires a new `collections` and `collection_images` table.

**Export selected**
Copy all selected images (or all images rated ≥ N stars) to a new folder on disk. Core to the culling workflow: pick → export for editing.

---

## Stats & Insights

**Dashboard**
Shots-per-day chart, rating distribution, tag frequency, and top camera/lens combinations. Useful for understanding a shoot at a glance.
