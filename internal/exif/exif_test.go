package exif

import (
	"os"
	"testing"
)

const (
	testCR2 = "../../internal/walker/TestData/RAW_CANON_EOS_1DX.CR2"
	testRAF = "../../internal/walker/TestData/RAW_FUJI_X-E1.RAF"
)

func TestExtractEXIF_NoEXIF(t *testing.T) {
	// A plain text file has no EXIF data.
	f, err := os.CreateTemp(t.TempDir(), "noexif*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("not a real jpeg")
	f.Close()

	_, err = ExtractEXIF(f.Name())
	if err == nil {
		t.Error("expected error for file without EXIF, got nil")
	}
}

func TestExtractEXIF_CR2(t *testing.T) {
	if _, err := os.Stat(testCR2); err != nil {
		t.Skip("test data not available: " + testCR2)
	}
	data, err := ExtractEXIF(testCR2)
	if err != nil {
		t.Fatalf("unexpected error extracting EXIF from CR2: %v", err)
	}
	if data.CaptureDate.IsZero() {
		t.Error("expected capture date to be set for CR2")
	}
}

func TestExtractEXIF_RAF(t *testing.T) {
	if _, err := os.Stat(testRAF); err != nil {
		t.Skip("test data not available: " + testRAF)
	}
	data, err := ExtractEXIF(testRAF)
	if err != nil {
		t.Fatalf("unexpected error extracting EXIF from RAF: %v", err)
	}
	if data.CaptureDate.IsZero() {
		t.Error("expected capture date to be set for RAF")
	}
}

func TestExtractEXIF_InvalidPath(t *testing.T) {
	_, err := ExtractEXIF("/nonexistent/file.jpg")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}
