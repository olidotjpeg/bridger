CREATE TABLE IF NOT EXISTS assets (
    id              TEXT PRIMARY KEY,
    path            TEXT NOT NULL UNIQUE,
    thumbnail_path  TEXT NOT NULL UNIQUE,
    file_size       INTEGER NOT NULL,
    modified_at     INTEGER NOT NULL,
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_assets_path ON assets(path);