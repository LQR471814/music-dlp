package main

import (
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/disintegration/imaging"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/vp8"
	_ "golang.org/x/image/vp8l"
	_ "golang.org/x/image/webp"
)

var urlRegex = regexp.MustCompile(`https?:\/\/`)

func CropImage(img image.Image) image.Image {
	x := img.Bounds().Dx()
	y := img.Bounds().Dy()
	side := x
	if y < x {
		side = y
	}
	cropped := imaging.CropCenter(img, side, side)
	return cropped
}

func AutoCompleteDirectory(input string) ([]string, error) {
	directory := filepath.Dir(input)
	files, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	entries := make([]string, len(files))
	for i, f := range files {
		entries[i] = filepath.Join(directory, f.Name())
	}
	return entries, nil
}

func GetCover(cover string, crop bool) (image.Image, string, error) {
	var coverSource io.Reader

	if urlRegex.MatchString(cover) {
		res, err := http.Get(cover)
		if err != nil {
			return nil, "", err
		}
		defer res.Body.Close()
		coverSource = res.Body
	} else {
		f, err := os.Open(cover)
		if err != nil {
			return nil, "", err
		}
		defer f.Close()
		coverSource = f
	}

	return image.Decode(coverSource)
}
