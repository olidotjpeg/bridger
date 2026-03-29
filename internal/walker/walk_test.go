package walk

import (
	"path/filepath"
	"testing"
)

func TestHasExtension(t *testing.T) {
	tests := []struct {
		path     string
		exts     []string
		expected bool
	}{
		{"image.png", []string{".png", ".jpg"}, true},
		{"image.gif", []string{".png", ".jpg"}, false},
		{"noextension", []string{".png", ".jpg"}, false},
	}

	for _, tt := range tests {
		got := hasExtension(tt.path, tt.exts)
		if got != tt.expected {
			t.Errorf("hasExtension(%q) = %v, want %v", tt.path, got, tt.expected)
		}
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
		if filepath.Ext(p) == ".mp4" {
			t.Errorf("unexpected .mp4 file in results: %s", p)
		}
	}
}
