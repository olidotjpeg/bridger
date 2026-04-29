package raw

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestIsRaw(t *testing.T) {
	tests := []struct {
		mimeType string
		expected bool
	}{
		{"image/x-canon-cr2", true},
		{"image/x-nikon-nef", true},
		{"image/x-sony-arw", true},
		{"image/x-fuji-raf", true},
		{"image/jpeg", false},
		{"image/png", false},
		{"", false},
		{"image/x-unknown-raw", false},
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			got := IsRaw(tt.mimeType)
			if got != tt.expected {
				t.Errorf("IsRaw(%q) = %v, want %v", tt.mimeType, got, tt.expected)
			}
		})
	}
}

func TestGeneratePreview_InvalidPath(t *testing.T) {
	_, err := GeneratePreview("/nonexistent/path/file.cr2", t.TempDir())
	if err == nil {
		t.Error("expected error for non-existent source file, got nil")
	}
}

// TestGeneratePreview_CacheSkip verifies that if the preview file already
// exists on disk, GeneratePreview returns its path immediately without
// attempting to re-generate it.
func TestGeneratePreview_CacheSkip(t *testing.T) {
	thumbDir := t.TempDir()
	srcPath := "/photos/test.cr2"

	// Compute the expected output path the same way GeneratePreview does.
	hash := fmt.Sprintf("%x", md5.Sum([]byte(srcPath)))
	cachedPath := filepath.Join(thumbDir, hash+"_preview.jpg")

	// Pre-create the file to simulate a previously cached result.
	if err := os.WriteFile(cachedPath, []byte("fake jpeg"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := GeneratePreview(srcPath, thumbDir)
	if err != nil {
		t.Fatalf("expected cache hit, got error: %v", err)
	}
	if got != cachedPath {
		t.Errorf("expected cached path %s, got %s", cachedPath, got)
	}
}
