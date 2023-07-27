package main

import (
	"fmt"
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
	execute(os.Args[1])
}

func execute(href string) error {
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
		meta, err := Prompt(info)
		if err != nil {
			log.Println(err)
			continue
		}

		meta.Write(tag)
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
