package main

import (
	"bytes"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"

	"github.com/bogem/id3v2/v2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("usage:\n%s YOUTUBE_LINK\n", os.Args[0])
		os.Exit(-1)
	}
	linkOrFile := os.Args[1]
	if !urlRegex.MatchString(linkOrFile) {
		err := edit(linkOrFile)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	download(linkOrFile)
}

func edit(file string) error {
	tag, err := id3v2.Open(
		file,
		id3v2.Options{Parse: true},
	)
	if err != nil {
		return err
	}
	defer tag.Close()

	pictureFrames := tag.GetFrames(tag.CommonID("Attached picture"))
	commentFrames := tag.GetFrames(tag.CommonID("Comment"))

	var cover id3v2.PictureFrame
	if len(pictureFrames) >= 1 {
		cover = pictureFrames[0].(id3v2.PictureFrame)
	}

	var comment id3v2.CommentFrame
	if len(commentFrames) >= 1 {
		comment = commentFrames[0].(id3v2.CommentFrame)
	}

	tagForm := &TagForm{
		Title:  tag.Title(),
		Artist: tag.Artist(),
		Album:  tag.Album(),
		Genre:  tag.Genre(),
		Year:   tag.Year(),
		Cover:  cover,
	}

	tagForm.PreviewImage, tagForm.PreviewImageMimetype, err = image.Decode(
		bytes.NewBuffer(cover.Picture),
	)
	if err != nil {
		return err
	}

	err = tagForm.EditUI("", false, comment.Text)
	if err != nil {
		return err
	}
	tagForm.Write(tag)

	return tag.Save()
}

func download(href string) error {
	log.Println("executing yt-dlp...")
	folderName, err := Download(href)
	if err != nil {
		return err
	}
	defer os.RemoveAll(folderName)
	files, err := os.ReadDir(folderName)
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		fullname := f.Name()
		audioPath := filepath.Join(folderName, fullname)
		name, ext := splitNameExtension(fullname)
		if ext == "json" {
			continue
		}
		if name == "" || ext == "" {
			log.Printf("invalid name or extension %s\n", fullname)
			continue
		}

		tag, err := id3v2.Open(
			audioPath,
			id3v2.Options{Parse: true},
		)
		if err != nil {
			log.Println(err)
			continue
		}
		defer tag.Close()

		info, err := ReadJsonInfo(
			filepath.Join(folderName, fmt.Sprintf("%s.info.json", name)),
		)
		if err != nil {
			log.Println(err)
			continue
		}

		tagForm := &TagForm{
			Title:  info.Title,
			Artist: info.Uploader,
			Album:  info.Title,
		}
		err = tagForm.EditUI(info.Thumbnail, true, info.Description)
		if err != nil {
			log.Println(err)
			continue
		}

		tagForm.Write(tag)
		err = tag.Save()
		if err != nil {
			log.Println(err)
			continue
		}

		err = os.Rename(audioPath, fullname)
		if err != nil {
			log.Println(err)
			continue
		}
	}

	return nil
}
