# Feature Request: Gallery Mode (Private Cloud Family Photo Gallery)

**Status:** Idea / not started
**Date:** 2026-06-12

## Summary

Add a second "personality" to Bridger: alongside the existing culling workflow, support a
browse-first **family photo gallery** for scanned physical photos, self-hosted in a homelab
and shareable with family.

The key architectural decision: this is **not** a second application or an extracted set of
reusable packages. It is the same binary, deployed as a second instance with a mode flag.

## Motivation

- I'm digitizing physical family photos (scanner dumps them into a directory).
- I want family members (e.g. on their phones) to browse them, grouped by people and events
  like "Summer 1994".
- Bridger's pipeline (walker → scanner → EXIF → thumbnails → SQLite → API) is exactly what
  this needs; only the workflow on top differs (browsing vs. culling).

## Decision: one binary, multiple instances

Considered and rejected:

- **Extract `internal/` packages into reusable libraries** — premature. ~4k lines of Go,
  single consumer. Within one Go module, a second entry point can already import everything;
  extraction buys nothing today.
- **Separate gallery application** — the data model and ingest pipeline are identical;
  forking duplicates maintenance.

Chosen approach: the existing static binary (`CGO_ENABLED=0`, embedded frontend, SQLite,
`--dir`/`--db`/`--thumbs` flags) is deployed **twice** in the homelab:

| Instance | Library | Mode |
|---|---|---|
| Culling | RAW working library | existing behavior |
| Family gallery | Scanned family photos | new gallery mode |

Two containers / systemd units, one codebase.

**Revisit if:** the gallery ever needs true multi-user accounts, per-person albums, or
upload-from-phone. That's the point where a separate entry point (e.g. `cmd/gallery`) starts
paying for itself. Until then, don't split.

## Proposed changes

### 1. Schema

- **Tag kinds** — add a `kind` column to `tags` (`person` | `keyword`) so people tags
  ("grandma", "grandpa") render differently from keyword tags ("summer").
- **Events / albums** — a grouping with a label and date ("Summer 1994"). Either generalize
  the existing `projects` tables (migration 7 is already a "group of photos" concept) or add
  a dedicated `events` table.
- **User-editable date on images** — important and easy to get wrong later: scanned photos
  carry the *scan* date in EXIF, not the real capture date. "Year" must be a manually set
  field on the image (or inherited from its event), not derived from EXIF.

### 2. Read-only / viewer mode

A `--readonly` flag (later: viewer role) that disables all mutating API routes. Family
browsing on a phone must not be able to delete or re-tag. All routes go through one Gin
router in `internal/api`, so this is a middleware, not a restructure.

### 3. Gallery UI

A browse-first front door in the same SPA: bigger thumbnails, grouped by event/year/person,
mobile-friendly. Lives as a route (e.g. `/gallery` vs `/cull`); the server picks the default
view from the mode flag.

### 4. Auth — punt initially

For "private cloud, share with family," rely on infrastructure the homelab already has:
Tailscale (family joins the tailnet) or a reverse proxy with Authelia/basic auth. In-app
auth only becomes worth building when roles are needed (admin can cull/tag, family is
read-only) — and the `--readonly` instance flag covers most of that in the meantime.

## Rough order of work

1. `--readonly` flag + route middleware (small, immediately useful)
2. Schema: tag kinds, events, user-set image date (+ migrations)
3. Gallery view in the SPA
4. Deploy second instance behind Tailscale/reverse proxy
5. (Later, if needed) in-app roles, separate entry point
