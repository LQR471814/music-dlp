package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randomString(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		sb.WriteByte(charset[rand.Intn(len(charset))])
	}
	return sb.String()
}

func splitNameExtension(filename string) (string, string) {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			if i+1 == len(filename) {
				return filename[:i], ""
			}
			return filename[:i], filename[i+1:]
		}
	}
	return "", ""
}

func Download(url string) (string, error) {
	folderName := randomString(10)
	cmd := exec.Command(
		"yt-dlp",
		"--write-info-json", "-x",
		"--audio-format", "mp3",
		"--sponsorblock-mark", "all",
		"-o", fmt.Sprintf("%s/%%(title)s.%%(ext)s", folderName),
		url,
	)

	_, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return folderName, nil
}

type JsonInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Thumbnail   string `json:"thumbnail"`
	Uploader    string `json:"uploader"`
	UploadDate  string `json:"upload_date"`
}

func ReadJsonInfo(filepath string) (JsonInfo, error) {
	var info JsonInfo
	f, err := os.Open(filepath)
	if err != nil {
		return info, err
	}
	decoder := json.NewDecoder(f)
	err = decoder.Decode(&info)
	return info, err
}
