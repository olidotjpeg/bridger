package db

import (
	"database/sql"
	"path/filepath"
	"runtime"
	"testing"

	walk "github.com/olidotjpeg/bridger/internal/walker"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	_, filename, _, _ := runtime.Caller(0)
	migrationsPath := filepath.Join(filepath.Dir(filename), "../../sql/migrations")

	if err := RunMigrations(db, migrationsPath); err != nil {
		t.Fatal(err)
	}

	return db
}

func TestUpsertImagePath_Insert(t *testing.T) {
	db := setupTestDB(t)

	file := walk.FileInfo{
		Path:     "/photos/image.png",
		FileName: "image.png",
		Size:     1000,
		MimeType: "image/png",
	}

	if err := UpsertImagePath(db, file); err != nil {
		t.Fatalf("unexpect error: %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM images WHERE file_path = ?", file.Path).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}
}

func TestUpsertImagePath_SkipUnchanged(t *testing.T) {
	db := setupTestDB(t)

	file := walk.FileInfo{
		Path:     "/photos/image.png",
		FileName: "image.png",
		Size:     1000,
		MimeType: "image/png",
	}

	UpsertImagePath(db, file)
	if err := UpsertImagePath(db, file); err != nil {
		t.Fatalf("unexpected error on second upsert: %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM images WHERE file_path = ?", file.Path).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}
}

func TestUpsertImagePath_UpdateChanged(t *testing.T) {
	db := setupTestDB(t)

	file := walk.FileInfo{
		Path:     "/photos/image.png",
		FileName: "image.png",
		Size:     1000,
		MimeType: "image/png",
	}

	UpsertImagePath(db, file)

	file.Size = 2000
	if err := UpsertImagePath(db, file); err != nil {
		t.Fatalf("unexpected error on update: %v", err)
	}

	var size int64
	db.QueryRow("SELECT file_size FROM images WHERE file_path = ?", file.Path).Scan(&size)
	if size != 2000 {
		t.Errorf("expected size 2000, got %d", size)
	}
}

func TestUpsertImagePath_PreservesRating(t *testing.T) {
	db := setupTestDB(t)

	file := walk.FileInfo{
		Path:     "/photos/image.png",
		FileName: "image.png",
		Size:     1000,
		MimeType: "image/png",
	}

	UpsertImagePath(db, file)
	db.Exec("UPDATE images SET rating = 5 WHERE file_path = ?", file.Path)

	file.Size = 2000
	UpsertImagePath(db, file)

	var rating int
	db.QueryRow("SELECT rating FROM images WHERE file_path = ?", file.Path).Scan(&rating)
	if rating != 5 {
		t.Errorf("expected rating 5 to be preserved, got %d", rating)
	}
}
