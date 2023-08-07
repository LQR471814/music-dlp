package main

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"log"
	"sync"
	"time"

	"github.com/bogem/id3v2/v2"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TagForm struct {
	Title                string
	Artist               string
	Album                string
	Genre                string
	Cover                id3v2.PictureFrame
	Year                 string
	PreviewImage         image.Image
	PreviewImageMimetype string

	ctx             context.Context
	cancel          func()
	imgLoadingMutex sync.Mutex
	croppedPreview  image.Image
}

func (f *TagForm) updatePreview(source string) bool {
	if f.cancel != nil {
		f.cancel()
	}
	f.ctx, f.cancel = context.WithTimeout(context.Background(), 500*time.Millisecond)
	<-f.ctx.Done()
	if !errors.Is(f.ctx.Err(), context.DeadlineExceeded) {
		return false
	}

	defer f.imgLoadingMutex.Unlock()
	f.imgLoadingMutex.Lock()
	coverImage, mimetype, err := GetCover(source, false)
	if err != nil {
		return false
	}

	f.PreviewImage = coverImage
	f.PreviewImageMimetype = mimetype
	f.croppedPreview = nil

	return true
}

func (f *TagForm) EditUI(previewSource string, cropImage bool, comments string) error {
	app := tview.NewApplication()

	previewBox := tview.NewImage()
	previewBox.SetBorder(true).SetTitle("cover preview")
	if previewSource != "" {
		f.updatePreview(previewSource)
	}
	if !cropImage {
		previewBox.SetImage(f.PreviewImage)
	} else {
		if f.croppedPreview == nil && f.PreviewImage != nil {
			f.croppedPreview = CropImage(f.PreviewImage)
		}
		previewBox.SetImage(f.croppedPreview)
	}

	cropImageCheck := tview.NewCheckbox().
		SetLabel("crop image").
		SetChecked(cropImage).
		SetChangedFunc(func(checked bool) {
			if !checked {
				previewBox.SetImage(f.PreviewImage)
				return
			}
			if f.croppedPreview == nil && f.PreviewImage != nil {
				f.croppedPreview = CropImage(f.PreviewImage)
			}
			previewBox.SetImage(f.croppedPreview)
		})

	coverField := tview.NewInputField().
		SetLabel("cover (filename/url)").
		SetText(previewSource).
		SetAutocompleteFunc(func(currentText string) (entries []string) {
			if currentText == "" {
				return nil
			}
			dirs, err := AutoCompleteDirectory(currentText)
			if err != nil {
				return nil
			}
			return dirs
		}).
		SetChangedFunc(func(text string) {
			if text == "" {
				return
			}
			go func() {
				changed := f.updatePreview(text)
				if changed {
					if cropImageCheck.IsChecked() && f.PreviewImage != nil {
						f.croppedPreview = CropImage(f.PreviewImage)
						previewBox.SetImage(f.croppedPreview)
					} else {
						previewBox.SetImage(f.PreviewImage)
					}
					app.Draw()
				}
			}()
		})

	coverField.SetInputCapture(WithResetInputCapture(coverField))

	title := tview.NewInputField().SetLabel("title").SetText(f.Title)
	artist := tview.NewInputField().SetLabel("artist").SetText(f.Artist)
	album := tview.NewInputField().SetLabel("album").SetText(f.Album)
	genre := tview.NewInputField().SetLabel("genre").SetText(f.Genre)
	year := tview.NewInputField().SetLabel("year").SetText(f.Year)

	title.SetInputCapture(WithResetInputCapture(title))
	artist.SetInputCapture(WithResetInputCapture(artist))
	album.SetInputCapture(WithResetInputCapture(album))
	genre.SetInputCapture(WithResetInputCapture(genre))
	year.SetInputCapture(WithResetInputCapture(year))

	form := tview.NewForm().
		AddFormItem(title).
		AddFormItem(artist).
		AddFormItem(album).
		AddFormItem(genre).
		AddFormItem(year).
		AddFormItem(coverField).
		AddFormItem(cropImageCheck)

	form.SetItemPadding(0)
	form.AddButton("save", func() {
		f.Title = title.GetText()
		f.Artist = artist.GetText()
		f.Album = album.GetText()
		f.Genre = genre.GetText()
		f.Year = year.GetText()

		coverSource := coverField.GetText()
		if coverSource != "" {
			defer f.imgLoadingMutex.Unlock()
			f.imgLoadingMutex.Lock()

			imgBuffer := bytes.NewBuffer(nil)

			targetImage := f.PreviewImage
			if cropImageCheck.IsChecked() {
				targetImage = f.croppedPreview
			}
			if targetImage != nil {
				err := png.Encode(imgBuffer, targetImage)
				if err != nil {
					log.Println(err)
				} else {
					f.Cover = id3v2.PictureFrame{
						MimeType:    "image/png",
						PictureType: id3v2.PTFrontCover,
						Encoding:    id3v2.EncodingUTF8,
						Picture:     imgBuffer.Bytes(),
					}
				}
			}
		}

		app.Stop()
	})
	form.AddButton("cancel", app.Stop)
	form.SetFocus(0)
	form.SetBorder(true)
	form.SetTitle("audio metadata")

	keyboardShortcuts := tview.NewTextView().
		SetText(
			"[ESC] or [CTRL] + [Q] discard changes and quit\n[CTRL] + [\\] clear current field\n[TAB] switch fields\n[CTRL] + [T] toggle focus between form and description",
		)
	keyboardShortcuts.SetBorder(true)
	keyboardShortcuts.SetTitle("keyboard shortcuts")

	commentView := tview.NewTextView().
		SetScrollable(true).
		SetText(comments)
	commentView.SetBorder(true)
	commentView.SetTitle("comments")

	leftColumn := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(keyboardShortcuts, 6, 1, false).
		AddItem(form, 0, 2, false).
		AddItem(commentView, 0, 3, false)

	layout := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(leftColumn, 0, 1, false).
		AddItem(previewBox, 0, 1, false)

	formFocused := true
	formFocusedPtr := &formFocused

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlT:
			if *formFocusedPtr {
				app.SetFocus(form)
			} else {
				app.SetFocus(commentView)
			}
			*formFocusedPtr = !*formFocusedPtr
		case tcell.KeyCtrlQ, tcell.KeyESC:
			app.Stop()
		}
		return event
	})

	return app.SetRoot(layout, true).SetFocus(form).Run()
}

func (f *TagForm) Write(tag *id3v2.Tag) {
	tag.DeleteAllFrames()
	tag.SetTitle(f.Title)
	tag.SetArtist(f.Artist)
	tag.SetAlbum(f.Album)
	tag.SetGenre(f.Genre)
	tag.SetYear(f.Year)
	tag.AddAttachedPicture(f.Cover)
}

func WithResetInputCapture(input *tview.InputField) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlBackslash:
			input.SetText("")
		}
		return event
	}
}
