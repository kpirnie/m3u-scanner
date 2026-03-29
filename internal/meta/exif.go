package meta

import (
	"fmt"
	"os"

	"github.com/kpirnie/m3u-scanner/internal/models"
	"github.com/rwcarlsen/goexif/exif"
)

// EnrichImage populates Year from EXIF DateTimeOriginal when available.
// Duration stays -1 for images — M3U players that support images expect this.
func EnrichImage(e *models.MediaEntry) {
	if e.Year != "" {
		return
	}

	year, err := exifYear(e.Path)
	if err == nil && year != "" {
		e.Year = year
	}
}

func exifYear(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return "", err
	}

	dt, err := x.DateTime()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d", dt.Year()), nil
}
