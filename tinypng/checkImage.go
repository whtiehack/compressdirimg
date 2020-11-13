package tinypng

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

// IsImage check is an image
func IsImage(filePath string) error {
	f, err := os.OpenFile(filePath, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, _, err = image.Decode(f)
	return err
}
