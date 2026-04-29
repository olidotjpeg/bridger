package walk

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHasExtension(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		exts     []string
		expected bool
	}{
		{"lowercase match", "image.png", []string{".png", ".jpg"}, true},
		{"uppercase PNG", "image.PNG", []string{".png", ".jpg"}, true},
		{"uppercase JPG", "image.JPG", []string{".png", ".jpg"}, true},
		{"unmatched extension", "image.gif", []string{".png", ".jpg"}, false},
		{"no extension", "noextension", []string{".png", ".jpg"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasExtension(tt.path, tt.exts)
			if got != tt.expected {
				t.Errorf("hasExtension(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestWalkDirectory(t *testing.T) {
	paths, err := WalkDirectory("./TestData", "")
	if err != nil {
		t.Fatal(err)
	}

	extensions := []string{".png", ".jpg", ".jpeg", ".cr2", ".nef", ".arw", ".raf"}
	expected := 0
	filepath.WalkDir("./TestData", func(path string, d fs.DirEntry, _ error) error {
		if !d.IsDir() && hasExtension(path, extensions) {
			expected++
		}
		return nil
	})

	if len(paths) != expected {
		t.Errorf("got %d paths, want %d", len(paths), expected)
	}

	for _, p := range paths {
		if strings.ToLower(p.MimeType) == "video/mp4" {
			t.Errorf("unexpected .mp4 file in results: %s", p.Path)
		}
		if p.FileName == "" {
			t.Errorf("empty FileName for %s", p.Path)
		}
		if p.MimeType == "" {
			t.Errorf("empty MimeType for %s", p.Path)
		}
		if p.Size == 0 {
			t.Errorf("zero Size for %s", p.Path)
		}
	}
}

func TestWalkDirectoryInvalidPath(t *testing.T) {
	paths, err := WalkDirectory("./nonexistent", "")
	if err == nil {
		t.Error("expected error for non-existent path, got nil")
	}
	if len(paths) != 0 {
		t.Errorf("expected empty paths, got %d", len(paths))
	}
}

func TestWalkDirectory_SkipsThumbDir(t *testing.T) {
	root := t.TempDir()
	thumbDir := filepath.Join(root, "thumbs")
	if err := os.MkdirAll(thumbDir, 0755); err != nil {
		t.Fatal(err)
	}

	// A real image in the root — should be returned.
	rootFile := filepath.Join(root, "photo.jpg")
	if err := os.WriteFile(rootFile, []byte("fake jpeg"), 0644); err != nil {
		t.Fatal(err)
	}

	// An image inside thumbDir — should be skipped.
	thumbFile := filepath.Join(thumbDir, "thumb.jpg")
	if err := os.WriteFile(thumbFile, []byte("fake thumb"), 0644); err != nil {
		t.Fatal(err)
	}

	paths, err := WalkDirectory(root, thumbDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, p := range paths {
		if p.Path == thumbFile {
			t.Errorf("thumb dir file should have been skipped, but got: %s", p.Path)
		}
	}

	found := false
	for _, p := range paths {
		if p.Path == rootFile {
			found = true
		}
	}
	if !found {
		t.Errorf("expected root file %s in results, but it was missing", rootFile)
	}
}

// --- StatFile ---

func TestStatFile_ValidFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "test*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("fake jpeg content")
	f.Close()

	info, err := StatFile(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Path != f.Name() {
		t.Errorf("expected Path %s, got %s", f.Name(), info.Path)
	}
	if info.FileName == "" {
		t.Error("expected non-empty FileName")
	}
	if info.MimeType == "" {
		t.Error("expected non-empty MimeType")
	}
	if info.Size == 0 {
		t.Error("expected non-zero Size")
	}
}

func TestStatFile_InvalidPath(t *testing.T) {
	_, err := StatFile("/nonexistent/path/image.jpg")
	if err == nil {
		t.Error("expected error for non-existent path, got nil")
	}
}

func TestStatFile_RawExtension(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "test*.cr2")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("fake raw content")
	f.Close()

	info, err := StatFile(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.MimeType != "image/x-canon-cr2" {
		t.Errorf("expected image/x-canon-cr2, got %s", info.MimeType)
	}
}