package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/olidotjpeg/bridger/internal/config"
	"github.com/olidotjpeg/bridger/internal/db"
	"github.com/olidotjpeg/bridger/internal/scanner"
	walk "github.com/olidotjpeg/bridger/internal/walker"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestRouter(t *testing.T) (*gin.Engine, *sql.DB) {
	t.Helper()

	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	_, filename, _, _ := runtime.Caller(0)
	migrationsPath := filepath.Join(filepath.Dir(filename), "../../sql/migrations")

	if err := db.RunMigrations(database, migrationsPath); err != nil {
		t.Fatal(err)
	}

	state := &scanner.ScanState{}
	reconfigCh := make(chan config.Config, 1)
	cfg := &config.Config{ScanDirs: []string{"."}, DBPath: ":memory:", ThumbsPath: t.TempDir()}
	router := SetupRouter(database, state, Config{
		ThumbDir:   cfg.ThumbsPath,
		NeedsSetup: false,
		CurrentCfg: cfg,
		ReconfigCh: reconfigCh,
	})

	t.Cleanup(func() { database.Close() })

	return router, database
}

func TestPing(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/ping", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetImages_Empty(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/images", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp PaginatedResponse[db.Image]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Total != 0 {
		t.Errorf("expected total 0, got %d", resp.Total)
	}
	if resp.Page != 1 {
		t.Errorf("expected page 1, got %d", resp.Page)
	}
}

func TestGetImages_Pagination(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/images?page=2&limit=10", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp PaginatedResponse[db.Image]
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Page != 2 {
		t.Errorf("expected page 2, got %d", resp.Page)
	}
	if resp.Limit != 10 {
		t.Errorf("expected limit 10, got %d", resp.Limit)
	}
}

func TestGetImageFull_NotFound(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/images/999/full", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetScanStatus(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/scan/status", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var status scanner.ScanStatus
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("failed to parse scan status: %v", err)
	}
}

func TestPostScan_Conflict(t *testing.T) {
	// Simulate a scan already running
	state := &scanner.ScanState{}
	state.TryStart()
	conflictRouter := SetupRouter(nil, state, Config{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/scan", nil)
	conflictRouter.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

// --- helpers ---

func seedRouterImage(t *testing.T, database *sql.DB, path string) string {
	t.Helper()
	file := walk.FileInfo{
		Path:     path,
		FileName: path[strings.LastIndex(path, "/")+1:],
		Size:     1000,
		MimeType: "image/jpeg",
	}
	if _, err := db.UpsertImagePath(database, file, "", ""); err != nil {
		t.Fatal(err)
	}
	var id string
	database.QueryRow("SELECT id FROM images WHERE file_path = ?", path).Scan(&id)
	return id
}

// --- GET /api/images filters ---

func TestGetImages_InvalidSortFallback(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/images?sort=malicious;DROP+TABLE+images", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetImages_RatingFilter(t *testing.T) {
	router, database := setupTestRouter(t)
	id := seedRouterImage(t, database, "/photos/a.jpg")
	seedRouterImage(t, database, "/photos/b.jpg")
	database.Exec("UPDATE images SET rating = 4 WHERE id = ?", id)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/images?rating=3", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp PaginatedResponse[db.Image]
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 1 {
		t.Errorf("expected total 1, got %d", resp.Total)
	}
}

// --- PATCH /api/images/:id ---

func TestPatchImage_Rating(t *testing.T) {
	router, database := setupTestRouter(t)
	id := seedRouterImage(t, database, "/photos/a.jpg")

	body := strings.NewReader(`{"rating": 5}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/images/"+id, body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var img db.Image
	json.Unmarshal(w.Body.Bytes(), &img)
	if img.Rating != 5 {
		t.Errorf("expected rating 5, got %d", img.Rating)
	}
}

func TestPatchImage_InvalidBody(t *testing.T) {
	router, database := setupTestRouter(t)
	id := seedRouterImage(t, database, "/photos/a.jpg")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/images/"+id, strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- GET /api/tags ---

func TestGetAllTags_Empty(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/tags", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var tags []db.Tag
	json.Unmarshal(w.Body.Bytes(), &tags)
	if tags == nil {
		t.Error("expected empty array, got null")
	}
}

// --- POST /api/tags ---

func TestPostTag_Created(t *testing.T) {
	router, _ := setupTestRouter(t)

	body := strings.NewReader(`{"name": "Wedding"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/tags", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}

	var tag db.Tag
	json.Unmarshal(w.Body.Bytes(), &tag)
	if tag.Name != "Wedding" {
		t.Errorf("expected name Wedding, got %s", tag.Name)
	}
	if tag.Id == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestPostTag_Conflict(t *testing.T) {
	router, _ := setupTestRouter(t)

	body := `{"name": "Wedding"}`
	for range 2 {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/tags", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/tags", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestPostTag_MissingName(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/tags", strings.NewReader(`{"name": ""}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- GET /api/images/:id/tags ---

func TestGetImageTags_Empty(t *testing.T) {
	router, database := setupTestRouter(t)
	id := seedRouterImage(t, database, "/photos/a.jpg")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/images/"+id+"/tags", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var tags []db.Tag
	json.Unmarshal(w.Body.Bytes(), &tags)
	if tags == nil {
		t.Error("expected empty array, got null")
	}
}

func TestGetImageTags_WithData(t *testing.T) {
	router, database := setupTestRouter(t)
	id := seedRouterImage(t, database, "/photos/a.jpg")

	// Create tags and associate them via PATCH
	tag1Body := strings.NewReader(`{"name": "Wedding"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/tags", tag1Body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	var tag1 db.Tag
	json.Unmarshal(w.Body.Bytes(), &tag1)

	patchBody := strings.NewReader(`{"tags": [` + strconv.Itoa(tag1.Id) + `]}`)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PATCH", "/api/images/"+id, patchBody)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/images/"+id+"/tags", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var tags []db.Tag
	json.Unmarshal(w.Body.Bytes(), &tags)
	if len(tags) != 1 {
		t.Errorf("expected 1 tag, got %d", len(tags))
	}
	if tags[0].Name != "Wedding" {
		t.Errorf("expected tag name Wedding, got %s", tags[0].Name)
	}
}

// --- POST /api/scan happy path ---

func TestPostScan_Accepted(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/scan", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", w.Code)
	}
}

// --- PATCH /api/images/:id with tags ---

func TestPatchImage_Tags(t *testing.T) {
	router, database := setupTestRouter(t)
	id := seedRouterImage(t, database, "/photos/a.jpg")

	// Create a tag first
	tagBody := strings.NewReader(`{"name": "Landscape"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/tags", tagBody)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	var tag db.Tag
	json.Unmarshal(w.Body.Bytes(), &tag)

	// Patch image with the tag
	patchBody := strings.NewReader(`{"tags": [` + strconv.Itoa(tag.Id) + `]}`)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PATCH", "/api/images/"+id, patchBody)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Verify via GET /api/images/:id/tags
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/images/"+id+"/tags", nil)
	router.ServeHTTP(w, req)

	var tags []db.Tag
	json.Unmarshal(w.Body.Bytes(), &tags)
	if len(tags) != 1 || tags[0].Name != "Landscape" {
		t.Errorf("expected Landscape tag after patch, got %v", tags)
	}
}

// --- GET /api/images/:id/full ---

func TestGetImageFull_JPEG(t *testing.T) {
	router, database := setupTestRouter(t)

	// Create a real temp file so c.File can serve it
	f, err := os.CreateTemp(t.TempDir(), "test*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("fake jpeg bytes")
	f.Close()

	id := seedRouterImage(t, database, f.Name())
	// Update the mime_type to image/jpeg explicitly
	database.Exec("UPDATE images SET mime_type = 'image/jpeg' WHERE id = ?", id)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/images/"+id+"/full", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		t.Errorf("expected image/* Content-Type, got %s", ct)
	}
}

func TestGetImageFull_RAW_WithPreview(t *testing.T) {
	router, database := setupTestRouter(t)

	// Create a temp preview JPEG that can be served
	previewFile, err := os.CreateTemp(t.TempDir(), "preview*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	previewFile.WriteString("fake preview jpeg")
	previewFile.Close()

	// Seed a RAW image record with a preview_path
	rawFile := walk.FileInfo{
		Path:     "/photos/raw.cr2",
		FileName: "raw.cr2",
		Size:     1000,
		MimeType: "image/x-canon-cr2",
	}
	db.UpsertImagePath(database, rawFile, "", previewFile.Name())

	var id string
	database.QueryRow("SELECT id FROM images WHERE file_path = ?", rawFile.Path).Scan(&id)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/images/"+id+"/full", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for RAW with preview, got %d", w.Code)
	}
}

func TestGetImageFull_RAW_NoPreview(t *testing.T) {
	router, database := setupTestRouter(t)

	// Seed a RAW image with no preview_path
	rawFile := walk.FileInfo{
		Path:     "/photos/raw_nopreview.cr2",
		FileName: "raw_nopreview.cr2",
		Size:     1000,
		MimeType: "image/x-canon-cr2",
	}
	db.UpsertImagePath(database, rawFile, "", "")

	var id string
	database.QueryRow("SELECT id FROM images WHERE file_path = ?", rawFile.Path).Scan(&id)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/images/"+id+"/full", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for RAW with no preview, got %d", w.Code)
	}
}

// --- GET /api/images sort/order variants ---

func TestGetImages_SortByFilename(t *testing.T) {
	router, database := setupTestRouter(t)
	seedRouterImage(t, database, "/photos/z.jpg")
	seedRouterImage(t, database, "/photos/a.jpg")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/images?sort=filename&order=asc", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp PaginatedResponse[db.Image]
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Data) < 2 {
		t.Fatalf("expected 2 images, got %d", len(resp.Data))
	}
	if resp.Data[0].Filename != "a.jpg" {
		t.Errorf("expected a.jpg first in asc filename order, got %s", resp.Data[0].Filename)
	}
}

func TestGetImages_InvalidOrderFallback(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/images?order=SIDEWAYS", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with safe fallback order, got %d", w.Code)
	}
}

// --- PATCH /api/images/:id not found ---

func TestPatchImage_NotFound(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/images/nonexistent-id", strings.NewReader(`{"rating": 3}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- GET /api/config ---

func TestGetConfig(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/config", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse config response: %v", err)
	}
	if _, ok := body["needs_setup"]; !ok {
		t.Error("expected needs_setup field in response")
	}
}

// --- PUT /api/config ---

func TestPutConfig_MissingBody(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/config", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPutConfig_InvalidDir(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/config", strings.NewReader(`{"scan_dirs": ["/nonexistent/path/xyz"]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-existent dir, got %d", w.Code)
	}
}

func TestPutConfig_ValidDir(t *testing.T) {
	dir := t.TempDir()

	state := &scanner.ScanState{}
	reconfigCh := make(chan config.Config, 1)
	cfg := &config.Config{ScanDirs: []string{}, DBPath: ":memory:", ThumbsPath: t.TempDir()}

	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })

	_, filename, _, _ := runtime.Caller(0)
	migrationsPath := filepath.Join(filepath.Dir(filename), "../../sql/migrations")
	if err := db.RunMigrations(database, migrationsPath); err != nil {
		t.Fatal(err)
	}

	router := SetupRouter(database, state, Config{
		ThumbDir:   cfg.ThumbsPath,
		NeedsSetup: true,
		CurrentCfg: cfg,
		ReconfigCh: reconfigCh,
	})

	body := `{"scan_dirs": ["` + dir + `"]}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Drain the channel so the goroutine doesn't block.
	<-reconfigCh
}

// --- GET /api/fs/list ---

func TestListDirectory_Default(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/fs/list", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if _, ok := body["path"]; !ok {
		t.Error("expected path field in response")
	}
	if _, ok := body["entries"]; !ok {
		t.Error("expected entries field in response")
	}
}

func TestListDirectory_SpecificPath(t *testing.T) {
	router, _ := setupTestRouter(t)
	dir := t.TempDir()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/fs/list?path="+dir, nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestListDirectory_InvalidPath(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/fs/list?path=/nonexistent/xyz/abc", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
