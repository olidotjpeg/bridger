package walk

import (
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
	paths, err := WalkDirectory("./TestData")
	if err != nil {
		t.Fatal(err)
	}

	if len(paths) != 6 {
		t.Errorf("got %d paths, want 6", len(paths))
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
	paths, err := WalkDirectory("./nonexistent")
	if err == nil {
		t.Error("expected error for non-existent path, got nil")
	}
	if len(paths) != 0 {
		t.Errorf("expected empty paths, got %d", len(paths))
	}
}