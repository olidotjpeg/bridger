package raw

import (
	"crypto/md5"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/davidbyttow/govips/v2/vips"
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

// GeneratePreview converts a RAW file to a full-resolution JPEG and saves it to
// previewDir. Returns the path to the saved JPEG. Skips generation if the file
// already exists.
func GeneratePreview(srcPath, previewDir string) (string, error) {
	hash := fmt.Sprintf("%x", md5.Sum([]byte(srcPath)))
	previewPath := filepath.Join(previewDir, hash+"_preview.jpg")

	if _, err := os.Stat(previewPath); err == nil {
		return previewPath, nil
	}

	buf, err := toJPEGVips(srcPath)
	if err != nil {
		buf, err = toJPEGExiftool(srcPath)
		if err != nil {
			return "", err
		}
	}

	if err := os.WriteFile(previewPath, buf, 0644); err != nil {
		return "", err
	}

	return previewPath, nil
}

func toJPEGVips(path string) ([]byte, error) {
	// NewThumbnailFromFile uses the libraw pipeline which handles RAF/NEF/CR2.
	// 10000px cap is above any camera sensor so this is effectively full resolution.
	img, err := vips.NewThumbnailFromFile(path, 10000, 0, vips.InterestingNone)
	if err != nil {
		return nil, err
	}
	defer img.Close()

	ep := vips.NewJpegExportParams()
	ep.Quality = 95
	buf, _, err := img.ExportJpeg(ep)
	return buf, err
}

func toJPEGExiftool(path string) ([]byte, error) {
	// LargePreviewImage is the highest quality embedded preview in most RAW files
	for _, tag := range []string{"-LargePreviewImage", "-JpgFromRaw", "-PreviewImage"} {
		out, err := exec.Command("exiftool", "-b", tag, path).Output()
		if err == nil && len(out) > 0 {
			return out, nil
		}
	}
	return nil, fmt.Errorf("no JPEG preview found in %s", path)
}
