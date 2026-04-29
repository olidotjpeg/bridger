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

	db, err := sql.Open("sqlite", ":memory:")
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

	if _, err := UpsertImagePath(db, file, "", ""); err != nil {
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

	UpsertImagePath(db, file, "", "")
	if _, err := UpsertImagePath(db, file, "", ""); err != nil {
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

	UpsertImagePath(db, file, "", "")

	file.Size = 2000
	if _, err := UpsertImagePath(db, file, "", ""); err != nil {
		t.Fatalf("unexpected error on update: %v", err)
	}

	var size int64
	db.QueryRow("SELECT file_size FROM images WHERE file_path = ?", file.Path).Scan(&size)
	if size != 2000 {
		t.Errorf("expected size 2000, got %d", size)
	}
}

func seedImage(t *testing.T, database *sql.DB, path string) string {
	t.Helper()
	file := walk.FileInfo{
		Path:     path,
		FileName: filepath.Base(path),
		Size:     1000,
		MimeType: "image/jpeg",
	}
	if _, err := UpsertImagePath(database, file, "", ""); err != nil {
		t.Fatal(err)
	}
	var id string
	database.QueryRow("SELECT id FROM images WHERE file_path = ?", path).Scan(&id)
	return id
}

func TestUpsertImagePath_PreservesRating(t *testing.T) {
	db := setupTestDB(t)

	file := walk.FileInfo{
		Path:     "/photos/image.png",
		FileName: "image.png",
		Size:     1000,
		MimeType: "image/png",
	}

	UpsertImagePath(db, file, "", "")
	db.Exec("UPDATE images SET rating = 5 WHERE file_path = ?", file.Path)

	file.Size = 2000
	UpsertImagePath(db, file, "", "")

	var rating int
	db.QueryRow("SELECT rating FROM images WHERE file_path = ?", file.Path).Scan(&rating)
	if rating != 5 {
		t.Errorf("expected rating 5 to be preserved, got %d", rating)
	}
}

// --- GetImagesWithCount ---

func TestGetImagesWithCount_Sort(t *testing.T) {
	db := setupTestDB(t)
	seedImage(t, db, "/photos/b.jpg")
	seedImage(t, db, "/photos/a.jpg")

	images, _, err := GetImagesWithCount(db, ImageQuery{Limit: 10, Sort: "filename", Order: "asc"})
	if err != nil {
		t.Fatal(err)
	}
	if images[0].Filename != "a.jpg" {
		t.Errorf("expected a.jpg first, got %s", images[0].Filename)
	}
}

func TestGetImagesWithCount_RatingFilter(t *testing.T) {
	db := setupTestDB(t)
	seedImage(t, db, "/photos/a.jpg")
	seedImage(t, db, "/photos/b.jpg")
	db.Exec("UPDATE images SET rating = 4 WHERE filename = 'a.jpg'")

	minRating := 3
	images, count, err := GetImagesWithCount(db, ImageQuery{Limit: 10, Sort: "filename", Order: "asc", MinRating: &minRating})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected filtered count 1, got %d", count)
	}
	if len(images) != 1 || images[0].Filename != "a.jpg" {
		t.Errorf("expected only a.jpg, got %v", images)
	}
}

func TestGetImagesWithCount_NoFilter_ReturnsAll(t *testing.T) {
	db := setupTestDB(t)
	seedImage(t, db, "/photos/a.jpg")
	seedImage(t, db, "/photos/b.jpg")

	_, count, err := GetImagesWithCount(db, ImageQuery{Limit: 10, Sort: "filename", Order: "asc"})
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
}

// --- Tags ---

func TestGetAllTags_Empty(t *testing.T) {
	db := setupTestDB(t)
	tags, err := GetAllTags(db)
	if err != nil {
		t.Fatal(err)
	}
	if tags == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(tags))
	}
}

func TestCreateTag(t *testing.T) {
	db := setupTestDB(t)
	tag, err := CreateTag(db, "Wedding")
	if err != nil {
		t.Fatal(err)
	}
	if tag.Name != "Wedding" {
		t.Errorf("expected name Wedding, got %s", tag.Name)
	}
	if tag.Id == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestCreateTag_Conflict(t *testing.T) {
	db := setupTestDB(t)
	CreateTag(db, "Wedding")
	_, err := CreateTag(db, "Wedding")
	if err == nil {
		t.Fatal("expected error on duplicate tag, got nil")
	}
	if !IsConflict(err) {
		t.Error("expected IsConflict to return true")
	}
}

func TestGetImageTags_Empty(t *testing.T) {
	db := setupTestDB(t)
	id := seedImage(t, db, "/photos/a.jpg")
	tags, err := GetImageTags(db, id)
	if err != nil {
		t.Fatal(err)
	}
	if tags == nil {
		t.Error("expected empty slice, got nil")
	}
}

// --- PatchImagesWithRatingOrTag ---

func TestPatchImage_UpdateRating(t *testing.T) {
	db := setupTestDB(t)
	id := seedImage(t, db, "/photos/a.jpg")

	rating := 4
	img, err := PatchImagesWithRatingOrTag(db, id, PatchImageInput{Rating: &rating})
	if err != nil {
		t.Fatal(err)
	}
	if img.Rating != 4 {
		t.Errorf("expected rating 4, got %d", img.Rating)
	}
}

func TestPatchImage_UpdateTags(t *testing.T) {
	db := setupTestDB(t)
	id := seedImage(t, db, "/photos/a.jpg")
	tag, _ := CreateTag(db, "Wedding")

	_, err := PatchImagesWithRatingOrTag(db, id, PatchImageInput{Tags: []int{tag.Id}})
	if err != nil {
		t.Fatal(err)
	}

	tags, err := GetImageTags(db, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 || tags[0].Name != "Wedding" {
		t.Errorf("expected Wedding tag, got %v", tags)
	}
}

func TestPatchImage_ClearTags(t *testing.T) {
	db := setupTestDB(t)
	id := seedImage(t, db, "/photos/a.jpg")
	tag, _ := CreateTag(db, "Wedding")
	PatchImagesWithRatingOrTag(db, id, PatchImageInput{Tags: []int{tag.Id}})

	_, err := PatchImagesWithRatingOrTag(db, id, PatchImageInput{Tags: []int{}})
	if err != nil {
		t.Fatal(err)
	}

	tags, _ := GetImageTags(db, id)
	if len(tags) != 0 {
		t.Errorf("expected 0 tags after clear, got %d", len(tags))
	}
}

func TestPatchImage_NilRatingSkipsUpdate(t *testing.T) {
	db := setupTestDB(t)
	id := seedImage(t, db, "/photos/a.jpg")
	db.Exec("UPDATE images SET rating = 5 WHERE id = ?", id)

	// Patch only tags — rating should be untouched
	_, err := PatchImagesWithRatingOrTag(db, id, PatchImageInput{Tags: []int{}})
	if err != nil {
		t.Fatal(err)
	}

	var rating int
	db.QueryRow("SELECT rating FROM images WHERE id = ?", id).Scan(&rating)
	if rating != 5 {
		t.Errorf("expected rating 5 to be preserved, got %d", rating)
	}
}

// --- GetImagePath ---

func TestGetImagePath_Found(t *testing.T) {
	db := setupTestDB(t)
	file := walk.FileInfo{
		Path:     "/photos/test.jpg",
		FileName: "test.jpg",
		Size:     1000,
		MimeType: "image/jpeg",
	}
	UpsertImagePath(db, file, "", "")

	var id string
	db.QueryRow("SELECT id FROM images WHERE file_path = ?", file.Path).Scan(&id)

	filePath, mimeType, previewPath, err := GetImagePath(db, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filePath != file.Path {
		t.Errorf("expected file_path %s, got %s", file.Path, filePath)
	}
	if mimeType != file.MimeType {
		t.Errorf("expected mime_type %s, got %s", file.MimeType, mimeType)
	}
	if previewPath != "" {
		t.Errorf("expected empty preview_path, got %s", previewPath)
	}
}

func TestGetImagePath_NotFound(t *testing.T) {
	db := setupTestDB(t)
	_, _, _, err := GetImagePath(db, "9999")
	if err == nil {
		t.Error("expected error for non-existent image, got nil")
	}
}

// --- GetImagesWithCount pagination and sort variants ---

func TestGetImagesWithCount_Pagination(t *testing.T) {
	db := setupTestDB(t)
	for i := 0; i < 5; i++ {
		seedImage(t, db, filepath.Join("/photos", filepath.Base(filepath.FromSlash(
			"/photos/"+string(rune('a'+i))+".jpg",
		))))
	}

	images, count, err := GetImagesWithCount(db, ImageQuery{Limit: 2, Offset: 2, Sort: "filename", Order: "asc"})
	if err != nil {
		t.Fatal(err)
	}
	if count != 5 {
		t.Errorf("expected total count 5, got %d", count)
	}
	if len(images) != 2 {
		t.Errorf("expected 2 images on page, got %d", len(images))
	}
}

func TestGetImagesWithCount_SortByRating(t *testing.T) {
	db := setupTestDB(t)
	idA := seedImage(t, db, "/photos/a.jpg")
	seedImage(t, db, "/photos/b.jpg")
	db.Exec("UPDATE images SET rating = 5 WHERE id = ?", idA)

	images, _, err := GetImagesWithCount(db, ImageQuery{Limit: 10, Sort: "rating", Order: "desc"})
	if err != nil {
		t.Fatal(err)
	}
	if len(images) < 2 {
		t.Fatalf("expected 2 images, got %d", len(images))
	}
	if images[0].Filename != "a.jpg" {
		t.Errorf("expected a.jpg (rating 5) first, got %s", images[0].Filename)
	}
}

func TestGetImagesWithCount_OrderAsc(t *testing.T) {
	db := setupTestDB(t)
	seedImage(t, db, "/photos/z.jpg")
	seedImage(t, db, "/photos/a.jpg")

	images, _, err := GetImagesWithCount(db, ImageQuery{Limit: 10, Sort: "filename", Order: "asc"})
	if err != nil {
		t.Fatal(err)
	}
	if len(images) < 2 {
		t.Fatalf("expected 2 images, got %d", len(images))
	}
	if images[0].Filename != "a.jpg" {
		t.Errorf("expected a.jpg first in asc order, got %s", images[0].Filename)
	}
}

// --- GetAllTags with data ---

func TestGetAllTags_WithData(t *testing.T) {
	db := setupTestDB(t)
	CreateTag(db, "Zebra")
	CreateTag(db, "Apple")
	CreateTag(db, "Mango")

	tags, err := GetAllTags(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(tags))
	}
	// Should be alphabetically ordered: Apple, Mango, Zebra
	if tags[0].Name != "Apple" {
		t.Errorf("expected Apple first (alphabetical), got %s", tags[0].Name)
	}
	if tags[2].Name != "Zebra" {
		t.Errorf("expected Zebra last, got %s", tags[2].Name)
	}
}

// --- GetImageTags with data ---

func TestGetImageTags_WithData(t *testing.T) {
	db := setupTestDB(t)
	id := seedImage(t, db, "/photos/a.jpg")
	tag1, _ := CreateTag(db, "Wedding")
	tag2, _ := CreateTag(db, "Landscape")
	PatchImagesWithRatingOrTag(db, id, PatchImageInput{Tags: []int{tag1.Id, tag2.Id}})

	tags, err := GetImageTags(db, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
	// Alphabetical: Landscape, Wedding
	if tags[0].Name != "Landscape" {
		t.Errorf("expected Landscape first, got %s", tags[0].Name)
	}
}

// --- PruneStaleEntries ---

func TestPruneStaleEntries_RemovesStale(t *testing.T) {
	db := setupTestDB(t)
	seedImage(t, db, "/photos/keep.jpg")
	seedImage(t, db, "/photos/gone.jpg")

	found := map[string]bool{"/photos/keep.jpg": true}
	pruned, err := PruneStaleEntries(db, []string{"/photos"}, found)
	if err != nil {
		t.Fatal(err)
	}
	if pruned != 1 {
		t.Errorf("expected 1 pruned, got %d", pruned)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM images").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 image remaining, got %d", count)
	}
}

func TestPruneStaleEntries_KeepsAll(t *testing.T) {
	db := setupTestDB(t)
	seedImage(t, db, "/photos/a.jpg")
	seedImage(t, db, "/photos/b.jpg")

	found := map[string]bool{"/photos/a.jpg": true, "/photos/b.jpg": true}
	pruned, err := PruneStaleEntries(db, []string{"/photos"}, found)
	if err != nil {
		t.Fatal(err)
	}
	if pruned != 0 {
		t.Errorf("expected 0 pruned, got %d", pruned)
	}
}

func TestPruneStaleEntries_NoDirs(t *testing.T) {
	db := setupTestDB(t)
	seedImage(t, db, "/photos/a.jpg")

	pruned, err := PruneStaleEntries(db, nil, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if pruned != 0 {
		t.Errorf("expected 0 pruned with no dirs, got %d", pruned)
	}
}
