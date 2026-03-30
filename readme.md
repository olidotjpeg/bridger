# Bridger

A CLI tool that scans a folder of photos and indexes them into a local SQLite database.

## Requirements

- Go 1.25+
- A C compiler (required by `go-sqlite3`) — on macOS this is provided by Xcode Command Line Tools:
  ```bash
  xcode-select --install
  ```
- `libvips` (required by `govips` for thumbnail generation):
  ```bash
  brew install vips
  ```

## Build

```bash
go build -o bridger ./cmd
```

## Run

```bash
./bridger --dir /path/to/photos --db ./bridger.db
```

### Flags

| Flag | Default | Description |
| --- | --- | --- |
| `--dir` | `.` | Root directory to scan for photos |
| `--db` | `./bridge.db` | Path to the SQLite database file |

### Example output

```
Scanning /photos/2024...
Found 6 files
  Inserted: 6
  Updated:  0
  Skipped:  0
  Errors:   0
Done.
```

Running again on the same folder will skip unchanged files:

```
Scanning /photos/2024...
Found 6 files
  Inserted: 0
  Updated:  0
  Skipped:  6
  Errors:   0
Done.
```

## Supported formats

`.jpg`, `.jpeg`, `.png` (case-insensitive)

## Test

```bash
go test ./...
```

## Project structure

```
bridger/
├── cmd/
│   └── main.go            # CLI entry point
├── internal/
│   ├── db/                # Database connection, migrations, upsert
│   ├── exif/              # EXIF metadata extraction
│   └── walker/            # Directory scanner
├── sql/
│   └── migrations/        # SQL migration files
└── TestData/              # Sample photos for local testing
```