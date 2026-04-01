package thumbs

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"

	"github.com/davidbyttow/govips/v2/vips"
)

func GenerateThumbnail(srcPath string, thumbDir string) (string, error) {
	hash := fmt.Sprintf("%x", md5.Sum([]byte(srcPath)))
	thumbPath := filepath.Join(thumbDir, hash+".jpg")

	if _, err := os.Stat(thumbPath); err == nil {
		return thumbPath, nil
	}

	image, err := vips.NewThumbnailFromFile(srcPath, 400, 0, vips.InterestingNone)

	if err != nil {
		return "", err
	}

	defer image.Close()

	ep := vips.NewJpegExportParams()
	buf, _, err := image.ExportJpeg(ep)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(thumbPath, buf, 0644); err != nil {
		return "", err
	}

	return thumbPath, nil
}
