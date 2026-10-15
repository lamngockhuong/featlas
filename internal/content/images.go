package content

import (
	"image"
	_ "image/jpeg" // register decoders for DecodeConfig
	_ "image/png"
	"os"

	_ "golang.org/x/image/webp"
)

// ImageInfo is the real size of an image file.
type ImageInfo struct {
	W     int   `json:"w"`
	H     int   `json:"h"`
	Bytes int64 `json:"bytes"`
}

// ReadImageInfo reads the pixel size from the image header.
func ReadImageInfo(path string) (ImageInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return ImageInfo{}, err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return ImageInfo{}, err
	}
	st, err := f.Stat()
	if err != nil {
		return ImageInfo{}, err
	}
	return ImageInfo{W: cfg.Width, H: cfg.Height, Bytes: st.Size()}, nil
}

// BoxInside reports whether box [x, y, w, h] lies inside the image.
func BoxInside(box []int, img ImageInfo) bool {
	return len(box) == 4 && box[0] >= 0 && box[1] >= 0 && box[2] > 0 && box[3] > 0 &&
		box[0]+box[2] <= img.W && box[1]+box[3] <= img.H
}
