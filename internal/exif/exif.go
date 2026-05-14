package exif

import (
	"fmt"
	"os"
	"time"

	goexif "github.com/rwcarlsen/goexif/exif"
)

type EXIFData struct {
	CaptureDate  time.Time
	Width        int
	Height       int
	CameraModel  string
	ISO          string
	Aperture     float32
	ShutterSpeed string
	FocalLength  float32
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

	if tag, err := x.Get(goexif.Model); err == nil {
		if val, err := tag.StringVal(); err == nil {
			data.CameraModel = val
		}
	}

	if tag, err := x.Get(goexif.ISOSpeedRatings); err == nil {
		if val, err := tag.StringVal(); err == nil {
			data.ISO = val
		}
	}

	if tag, err := x.Get(goexif.FNumber); err == nil {
		if val, err := tag.Rat(0); err == nil {
			f, _ := val.Float64()
			data.Aperture = float32(f)
		}
	}

	if tag, err := x.Get(goexif.ExposureTime); err == nil {
		if val, err := tag.Rat(0); err == nil {
			data.ShutterSpeed = fmt.Sprintf("%d/%d", val.Num(), val.Denom())
		}
	}

	if tag, err := x.Get(goexif.FocalLength); err == nil {
		if val, err := tag.Rat(0); err == nil {
			f, _ := val.Float64()
			data.FocalLength = float32(f)
		}
	}

	return data, nil
}
