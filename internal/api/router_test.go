package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/olidotjpeg/bridger/internal/db"
	"github.com/olidotjpeg/bridger/internal/scanner"
	walk "github.com/olidotjpeg/bridger/internal/walker"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestRouter(t *testing.T) (*gin.Engine, *sql.DB) {
	t.Helper()

	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	_, filename, _, _ := runtime.Caller(0)
	migrationsPath := filepath.Join(filepath.Dir(filename), "../../sql/migrations")

	if err := db.RunMigrations(database, migrationsPath); err != nil {
		t.Fatal(err)
	}

	state := &scanner.ScanState{}
	router := SetupRouter(database, state, Config{WalkDir: ".", ThumbDir: t.TempDir()})

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
	if _, err := db.UpsertImagePath(database, file, ""); err != nil {
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
