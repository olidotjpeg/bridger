package exif

import (
	"testing"
)

const (
	testJPEG = "../../internal/walker/TestData/2024-12/VRChat_2024-12-06_20-52-55.177_1920x1080.png"
	testCR2  = "../../internal/walker/TestData/RAW_CANON_EOS_1DX.CR2"
	testRAF  = "../../internal/walker/TestData/RAW_FUJI_X-E1.RAF"
)

func TestExtractEXIF_NoEXIF(t *testing.T) {
	// VRChat PNGs have no EXIF data
	_, err := ExtractEXIF(testJPEG)
	if err == nil {
		t.Error("expected error for file without EXIF, got nil")
	}
}

func TestExtractEXIF_CR2(t *testing.T) {
	data, err := ExtractEXIF(testCR2)
	if err != nil {
		t.Fatalf("unexpected error extracting EXIF from CR2: %v", err)
	}
	if data.CaptureDate.IsZero() {
		t.Error("expected capture date to be set for CR2")
	}
}

func TestExtractEXIF_RAF(t *testing.T) {
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
