CREATE TABLE IF NOT EXISTS images (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_path TEXT NOT NULL UNIQUE,
    filename TEXT NOT NULL,
    file_size INTEGER,
    mime_type TEXT,
    thumbnail_path TEXT,
    rating INTEGER DEFAULT 0,
    capture_date DATETIME,
    width INTEGER,
    height INTEGER,
    index_at DATETIME NOT NULL
);