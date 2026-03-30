package thumbs

import (
	"github.com/davidbyttow/govips/v2/vips"
)

func GenerateThumbnail(srcPath string, thumbDir string) (string, error) {
	image, err := vips.NewThumbnailFromFile(srcPath, 200, 200, vips.InterestingAttention)

	if err != nil {
		return "", err
	}

	return "", nil
}
