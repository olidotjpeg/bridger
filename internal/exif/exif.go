package exif

import (
	"os"
	"time"

	goexif "github.com/rwcarlsen/goexif/exif"
)

type EXIFData struct {
	CaptureDate time.Time
	Width       int
	Height      int
}

func ExtractEXIF(path string) (*EXIFData, error) {
	f, err := os.Open(path)

	if err != nil {
		return nil, err
	}
	defer f.Close()

	x, err := goexif.Decode(f)

	if err != nil {
		return nil, err
	}
	data := &EXIFData{}

	if date, err := x.DateTime(); err == nil {
		data.CaptureDate = date
	}

	if tag, err := x.Get(goexif.PixelXDimension); err == nil {
		if val, err := tag.Int(0); err == nil {
			data.Width = val
		}
	}

	if tag, err := x.Get(goexif.PixelYDimension); err == nil {
		if val, err := tag.Int(0); err == nil {
			data.Height = val
		}
	}

	return data, nil
}
