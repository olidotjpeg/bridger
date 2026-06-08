package raw

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"image/jpeg"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
	goexif "github.com/rwcarlsen/goexif/exif"
)

var mimeTypes = map[string]bool{
	"image/x-canon-cr2": true,
	"image/x-nikon-nef": true,
	"image/x-sony-arw":  true,
	"image/x-fuji-raf":  true,
}

// IsRaw reports whether the given MIME type is a RAW camera format.
func IsRaw(mimeType string) bool {
	return mimeTypes[mimeType]
}

// GeneratePreview extracts the embedded JPEG preview from a RAW file and saves
// it to previewDir. Returns the path to the saved JPEG. Skips if already cached.
func GeneratePreview(srcPath, previewDir string) (string, error) {
	hash := fmt.Sprintf("%x", md5.Sum([]byte(srcPath)))
	previewPath := filepath.Join(previewDir, hash+"_preview.jpg")

	if _, err := os.Stat(previewPath); err == nil {
		return previewPath, nil
	}

	buf, err := extractEmbeddedJPEG(srcPath)
	if err != nil {
		return "", err
	}

	buf = applyRawOrientation(srcPath, buf)

	if err := os.WriteFile(previewPath, buf, 0644); err != nil {
		return "", err
	}

	return previewPath, nil
}

// extractEmbeddedJPEG finds the highest-resolution JPEG embedded in a RAW file.
// All common RAW formats (CR2, NEF, ARW, RAF) contain embedded camera-generated
// JPEG previews. We use jpeg.Decode for exact boundary detection and pick the
// candidate with the largest pixel area (not byte size, since an EXIF thumbnail
// inside the large preview JPEG can fool byte-count heuristics).
func extractEmbeddedJPEG(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	soiMarker := []byte{0xFF, 0xD8, 0xFF}

	var bestData []byte
	bestPixels := 0

	pos := 0
	for pos < len(data) {
		idx := bytes.Index(data[pos:], soiMarker)
		if idx < 0 {
			break
		}
		soi := pos + idx

		// Decode the JPEG starting at this SOI. jpeg.Decode advances the reader
		// exactly to the EOI, so r.Len() tells us how many bytes remain after the JPEG.
		r := bytes.NewReader(data[soi:])
		img, decErr := jpeg.Decode(r)
		if decErr == nil {
			b := img.Bounds()
			pixels := b.Dx() * b.Dy()
			if pixels > bestPixels {
				consumed := len(data[soi:]) - r.Len()
				bestData = data[soi : soi+consumed]
				bestPixels = pixels
			}
		}

		pos = soi + 1
	}

	if len(bestData) == 0 {
		return nil, fmt.Errorf("no embedded JPEG preview found in %s", path)
	}

	out := make([]byte, len(bestData))
	copy(out, bestData)
	return out, nil
}

// applyRawOrientation reads the EXIF orientation from the RAW container and
// physically rotates the extracted JPEG preview to match. Embedded JPEG previews
// in RAW files often lack their own orientation tag — the rotation is stored in
// the parent RAW's EXIF instead. Falls back to returning the original bytes on
// any error so the preview is always usable even if orientation is unknown.
func applyRawOrientation(rawPath string, jpegBuf []byte) []byte {
	f, err := os.Open(rawPath)
	if err != nil {
		return jpegBuf
	}
	defer f.Close()

	x, err := goexif.Decode(f)
	if err != nil {
		return jpegBuf
	}

	tag, err := x.Get(goexif.Orientation)
	if err != nil {
		return jpegBuf
	}

	orient, err := tag.Int(0)
	if err != nil || orient <= 1 {
		return jpegBuf
	}

	img, err := jpeg.Decode(bytes.NewReader(jpegBuf))
	if err != nil {
		return jpegBuf
	}

	switch orient {
	case 2:
		img = imaging.FlipH(img)
	case 3:
		img = imaging.Rotate180(img)
	case 4:
		img = imaging.FlipV(img)
	case 5:
		img = imaging.Transpose(img)
	case 6:
		img = imaging.Rotate270(img)
	case 7:
		img = imaging.Transverse(img)
	case 8:
		img = imaging.Rotate90(img)
	default:
		return jpegBuf
	}

	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: 92}); err != nil {
		return jpegBuf
	}
	return out.Bytes()
}
