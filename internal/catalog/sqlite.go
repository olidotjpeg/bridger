package catalog

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStore struct {
	db *sql.DB
}

func OpenSQLite(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", path)

	if err != nil {
		return nil, err
	}

	// Pragmas for performance and safety
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA foreign_keys=ON;",
	}

	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return nil, err
		}
	}

	if err := migrate(db); err != nil {
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS assets (
		id           TEXT PRIMARY KEY,
		path         TEXT NOT NULL UNIQUE,
		file_size    INTEGER NOT NULL,
		modified_at  INTEGER NOT NULL,
		created_at   INTEGER NOT NULL,
		updated_at   INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_assets_path ON assets(path);
	`

	_, err := db.Exec(schema)

	return err
}
