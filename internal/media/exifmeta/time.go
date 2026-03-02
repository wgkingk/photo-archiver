package exifmeta

import (
	"os"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

func ShotAt(path string, fallback time.Time) (time.Time, bool) {
	f, err := os.Open(path)
	if err != nil {
		return fallback, false
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return fallback, false
	}

	dt, err := x.DateTime()
	if err == nil && !dt.IsZero() {
		return dt, true
	}

	tag, err := x.Get(exif.DateTimeOriginal)
	if err != nil {
		return fallback, false
	}
	val, err := tag.StringVal()
	if err != nil {
		return fallback, false
	}
	val = strings.Trim(val, "\x00 ")
	parsed, err := time.Parse("2006:01:02 15:04:05", val)
	if err != nil {
		return fallback, false
	}
	return parsed, true
}
