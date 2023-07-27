package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bogem/id3v2/v2"
	"github.com/disintegration/imaging"
	"github.com/gabriel-vasile/mimetype"
)

var urlRegex = regexp.MustCompile(`https?:\/\/`)

type Metadata struct {
	Title        string
	Artist       string
	Album        string
	Cover        string // this is a url or filename
	Year         int
	CropToSquare bool
}

func (m Metadata) Write(tag *id3v2.Tag) {
	tag.SetTitle(m.Title)
	tag.SetArtist(m.Artist)
	tag.SetAlbum(m.Album)
	tag.SetYear(fmt.Sprint(m.Year))
	picture, err := GetCover(m.Cover, m.CropToSquare)
	if err == nil {
		tag.AddAttachedPicture(picture)
	} else {
		log.Println(err)
	}
}

func GetCover(cover string, crop bool) (id3v2.PictureFrame, error) {
	var coverSource io.Reader

	if urlRegex.MatchString(cover) {
		res, err := http.Get(cover)
		if err != nil {
			return id3v2.PictureFrame{}, err
		}
		defer res.Body.Close()
		coverSource = res.Body
	} else {
		f, err := os.Open(cover)
		if err != nil {
			return id3v2.PictureFrame{}, err
		}
		defer f.Close()
		coverSource = f
	}

	contents, err := io.ReadAll(coverSource)
	if err != nil {
		return id3v2.PictureFrame{}, err
	}
	mime := mimetype.Detect(contents)
	mimestring := mime.String()

	if !strings.HasPrefix(mimestring, "image") {
		return id3v2.PictureFrame{}, fmt.Errorf("invalid image format %s", mimestring)
	}

	if crop {
		cropped, filetype, err := CropImage(contents)
		if err != nil {
			log.Println("failed crop, fallback to non-cropped image:", err.Error())
		} else {
			contents = cropped
			mimestring = filetype
		}
	}

	return id3v2.PictureFrame{
		Encoding:    id3v2.EncodingUTF8,
		MimeType:    mimestring,
		Picture:     contents,
		PictureType: id3v2.PTFrontCover,
	}, nil
}

func CropImage(contents []byte) ([]byte, string, error) {
	img, _, err := image.Decode(bytes.NewBuffer(contents))
	if err != nil {
		return nil, "", err
	}

	x := img.Bounds().Dx()
	y := img.Bounds().Dy()
	side := x
	if y < x {
		side = y
	}

	cropped := imaging.CropCenter(img, side, side)
	buffer := bytes.NewBuffer(nil)
	err = png.Encode(buffer, cropped)
	if err != nil {
		return nil, "", err
	}
	return buffer.Bytes(), "image/png", nil
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

func Prompt(info JsonInfo) (Metadata, error) {
	fmt.Printf(
		"===== Youtube description =====\n%s\n===============================\n",
		info.Description,
	)

	var title string
	err := survey.AskOne(
		&survey.Input{
			Default: info.Title,
			Message: "track title",
			Help:    "the track title.",
		},
		&title,
	)
	if err != nil {
		return Metadata{}, err
	}

	var artist string
	err = survey.AskOne(
		&survey.Input{
			Default: info.Uploader,
			Message: "artist",
			Help:    "the track artist (a single person/entity).",
		},
		&artist,
	)
	if err != nil {
		return Metadata{}, err
	}

	var album string
	err = survey.AskOne(
		&survey.Input{
			Message: "album",
			Default: title,
			Help:    "the name of the album.",
		},
		&album,
	)
	if err != nil {
		return Metadata{}, err
	}

	var cover string
	err = survey.AskOne(
		&survey.Input{
			Message: "track cover",
			Default: info.Thumbnail,
			Suggest: func(toComplete string) []string {
				directories, err := AutoCompleteDirectory(toComplete)
				if err != nil {
					return nil
				}
				return directories
			},
			Help: "the front cover of the track.",
		},
		&cover,
	)
	if err != nil {
		return Metadata{}, err
	}

	var yearString string
	err = survey.AskOne(
		&survey.Input{
			Message: "year",
			Default: info.UploadDate[:4],
			Help:    "the year the track was released.",
		},
		&yearString,
	)
	if err != nil {
		return Metadata{}, err
	}

	var cropSquare string
	err = survey.AskOne(
		&survey.Select{
			Message: "crop cover to square?",
			Options: []string{"yes", "no"},
		},
		&cropSquare,
	)
	if err != nil {
		return Metadata{}, err
	}

	year, err := strconv.Atoi(yearString)
	if err != nil {
		return Metadata{}, err
	}

	return Metadata{
		Title:        title,
		Artist:       artist,
		Album:        album,
		Cover:        cover,
		Year:         year,
		CropToSquare: cropSquare == "yes",
	}, nil
}
