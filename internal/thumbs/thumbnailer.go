package thumbs

import (
	"crypto/md5"
	"fmt"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
)

func GenerateThumbnail(srcPath string, thumbDir string) (string, error) {
	hash := fmt.Sprintf("%x", md5.Sum([]byte(srcPath)))
	thumbPath := filepath.Join(thumbDir, hash+".jpg")

	if _, err := os.Stat(thumbPath); err == nil {
		return thumbPath, nil
	}

	src, err := imaging.Open(srcPath, imaging.AutoOrientation(true))
	if err != nil {
		return "", err
	}

	thumb := imaging.Fit(src, 400, 400, imaging.Lanczos)

	out, err := os.Create(thumbPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if err := jpeg.Encode(out, thumb, &jpeg.Options{Quality: 85}); err != nil {
		out.Close()
		os.Remove(thumbPath)
		return "", err
	}

	return thumbPath, nil
}
