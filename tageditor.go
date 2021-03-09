// Copyright (C) 2020  Raziman

package main

import (
	"errors"
	"fmt"
	"sort"

	"github.com/bogem/id3v2"
	"github.com/gdamore/tcell/v2"
	"github.com/issadarkthing/gomu/lyric"
	"github.com/rivo/tview"
	"github.com/ztrue/tracerr"
)

type myFlex struct {
	*tview.Flex
	FocusedItem tview.Primitive
}

func tagPopup(node *AudioFile) (err error) {

	popupID := "tag-editor-input-popup"
	tag, popupLyricMap, options, err := loadTagMap(node)
	if err != nil {
		return tracerr.Wrap(err)
	}

	var (
		artistInputField  *tview.InputField = tview.NewInputField()
		titleInputField   *tview.InputField = tview.NewInputField()
		albumInputField   *tview.InputField = tview.NewInputField()
		getTagButton      *tview.Button     = tview.NewButton("Get Tag")
		saveTagButton     *tview.Button     = tview.NewButton("Save Tag")
		lyricDropDown     *tview.DropDown   = tview.NewDropDown()
		deleteLyricButton *tview.Button     = tview.NewButton("Delete Lyric")
		getLyricDropDown  *tview.DropDown   = tview.NewDropDown()
		getLyricButton    *tview.Button     = tview.NewButton("Fetch Lyric")
		lyricTextView     *tview.TextView
		leftGrid          *tview.Grid = tview.NewGrid()
		rightFlex         *tview.Flex = tview.NewFlex()
	)
	artistInputField.SetLabel("Artist: ").
		SetFieldWidth(20).
		SetText(tag.Artist()).
		SetFieldBackgroundColor(gomu.colors.popup).
		SetBackgroundColor(gomu.colors.background)
	titleInputField.SetLabel("Title:  ").
		SetFieldWidth(20).
		SetText(tag.Title()).
		SetFieldBackgroundColor(gomu.colors.popup).
		SetBackgroundColor(gomu.colors.background)
	albumInputField.SetLabel("Album:  ").
		SetFieldWidth(20).
		SetText(tag.Album()).
		SetFieldBackgroundColor(gomu.colors.popup).
		SetBackgroundColor(gomu.colors.background)
	getTagButton.SetSelectedFunc(func() {
		var titles []string
		audioFile := node
		lang := "zh-CN"
		results, err := lyric.GetLyricOptions(lang, audioFile.name)
		if err != nil {
			errorPopup(err)
			return
		}
		for _, v := range results {
			titles = append(titles, v.TitleForPopup)
		}

		searchPopup(" Song Tags ", titles, func(selected string) {
			if selected == "" {
				return
			}

			var selectedIndex int
			for i, v := range results {
				if v.TitleForPopup == selected {
					selectedIndex = i
					break
				}
			}

			newTag := results[selectedIndex]
			artistInputField.SetText(newTag.Artist)
			titleInputField.SetText(newTag.Title)
			albumInputField.SetText(newTag.Album)

			tag, err = id3v2.Open(node.path, id3v2.Options{Parse: true})
			if err != nil {
				errorPopup(err)
			}
			defer tag.Close()
			tag.SetArtist(newTag.Artist)
			tag.SetTitle(newTag.Title)
			tag.SetAlbum(newTag.Album)
			err = tag.Save()
			if err != nil {
				errorPopup(err)
			} else {
				defaultTimedPopup(" Success ", "Tag update successfully")
			}
		})
	}).
		SetBorder(true).
		SetBackgroundColor(gomu.colors.background).
		SetTitleColor(gomu.colors.accent)
	saveTagButton.SetSelectedFunc(func() {
		tag, err = id3v2.Open(node.path, id3v2.Options{Parse: true})
		if err != nil {
			errorPopup(err)
		}
		defer tag.Close()
		tag.SetArtist(artistInputField.GetText())
		tag.SetTitle(titleInputField.GetText())
		tag.SetAlbum(albumInputField.GetText())
		err = tag.Save()
		if err != nil {
			errorPopup(err)
		} else {
			defaultTimedPopup(" Success ", "Tag update successfully")
		}
	}).
		SetBorder(true).
		SetBackgroundColor(gomu.colors.background).
		SetTitleColor(gomu.colors.foreground)

	lyricDropDown.SetOptions(options, nil).
		SetCurrentOption(0).
		SetFieldBackgroundColor(gomu.colors.background).
		SetFieldTextColor(gomu.colors.accent).
		SetPrefixTextColor(gomu.colors.accent).
		SetSelectedFunc(func(text string, _ int) {
			lyricTextView.SetText(popupLyricMap[text]).
				SetTitle(" " + text + " lyric preview ")
		}).
		SetLabel("Embeded Lyrics: ")
	lyricDropDown.SetBackgroundColor(gomu.colors.popup)

	deleteLyricButton.SetSelectedFunc(func() {
		_, langExt := lyricDropDown.GetCurrentOption()
		if len(options) > 0 {
			err := embedLyric(node.path, "", langExt, true)
			if err != nil {
				errorPopup(err)
			} else {
				infoPopup(langExt + " lyric deleted successfully.")
			}
			// Update map
			delete(popupLyricMap, langExt)

			// Update dropdown options
			var newOptions []string
			for _, v := range options {
				if v == langExt {
					continue
				}
				newOptions = append(newOptions, v)
			}
			options = newOptions
			lyricDropDown.SetOptions(newOptions, nil).
				SetCurrentOption(0).
				SetSelectedFunc(func(text string, _ int) {
					lyricTextView.SetText(popupLyricMap[text]).
						SetTitle(" " + text + " lyric preview ")
				})

				// Update lyric preview
			if len(newOptions) > 0 {
				_, langExt = lyricDropDown.GetCurrentOption()
				lyricTextView.SetText(popupLyricMap[langExt]).
					SetTitle(" " + langExt + " lyric preview ")
			} else {
				langExt = ""
				lyricTextView.SetText("No lyric embeded.").
					SetTitle(" " + langExt + " lyric preview ")
			}
		} else {
			infoPopup("No lyric embeded.")
		}
	}).
		SetBorder(true).
		SetBackgroundColor(gomu.colors.background).
		SetTitleColor(gomu.colors.accent)

	getLyricDropDownOptions := []string{"en", "zh-CN"}
	getLyricDropDown.SetOptions(getLyricDropDownOptions, nil).
		SetCurrentOption(0).
		SetFieldBackgroundColor(gomu.colors.background).
		SetFieldTextColor(gomu.colors.accent).
		SetPrefixTextColor(gomu.colors.accent).
		SetLabel("Fetch Lyrics: ")
	lyricDropDown.SetBackgroundColor(gomu.colors.popup)

	getLyricButton.SetSelectedFunc(func() {

		audioFile := gomu.playlist.getCurrentFile()
		_, lang := getLyricDropDown.GetCurrentOption()
		if audioFile.isAudioFile {
			go func() {
				gomu.app.QueueUpdateDraw(func() {

					var titles []string
					results, err := lyric.GetLyricOptions(lang, audioFile.name)
					if err != nil {
						errorPopup(err)
						gomu.app.Draw()
					}

					for _, v := range results {
						titles = append(titles, v.TitleForPopup)
					}

					searchPopup(" Lyrics ", titles, func(selected string) {
						if selected == "" {
							return
						}

						go func() {
							var selectedIndex int
							for i, v := range results {
								if v.TitleForPopup == selected {
									selectedIndex = i
									break
								}
							}
							lyric, err := lyric.GetLyric(results[selectedIndex].LangExt, results[selectedIndex])
							if err != nil {
								errorPopup(err)
								gomu.app.Draw()
							}

							err = embedLyric(audioFile.path, lyric, lang, false)
							if err != nil {
								errorPopup(err)
								gomu.app.Draw()
							} else {
								infoPopup(lang + " lyric added successfully")
								gomu.app.Draw()
							}

							// This is to ensure that the above go routine finish.
							_, popupLyricMap, newOptions, err := loadTagMap(audioFile)
							if err != nil {
								errorPopup(err)
								gomu.app.Draw()
								return
							}

							options = newOptions
							// Update dropdown options
							lyricDropDown.SetOptions(newOptions, nil).
								SetCurrentOption(0).
								SetSelectedFunc(func(text string, _ int) {
									lyricTextView.SetText(popupLyricMap[text]).
										SetTitle(" " + text + " lyric preview ")
								})

							// Update lyric preview
							if len(newOptions) > 0 {
								_, langExt := lyricDropDown.GetCurrentOption()
								lyricTextView.SetText(popupLyricMap[langExt]).
									SetTitle(" " + langExt + " lyric preview ")
							} else {
								lyricTextView.SetText("No lyric embeded.").
									SetTitle(" lyric preview ")
							}

						}()
					})

				})
			}()
		} else {
			errorPopup(errors.New("not an audio file"))
			gomu.app.Draw()
		}
	}).
		SetBorder(true).
		SetBackgroundColor(gomu.colors.background).
		SetTitleColor(gomu.colors.accent)

	var lyricText string
	_, langExt := lyricDropDown.GetCurrentOption()
	lyricText = popupLyricMap[langExt]
	if lyricText == "" {
		lyricText = "No lyric embeded."
		langExt = ""
	}
	lyricTextView = tview.NewTextView()
	lyricTextView.
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true).
		SetTitle(" " + langExt + " lyric preview ").
		SetBorder(true)

	lyricTextView.SetText(lyricText).
		SetScrollable(true).
		SetWordWrap(true).
		SetWrap(true).
		SetBackgroundColor(gomu.colors.popup).
		SetBorder(true)

	leftGrid.SetRows(3, 3, 3, 3, 3, 0, 3, 3, 3, 3, 3).
		SetColumns(30).
		AddItem(artistInputField, 0, 0, 1, 3, 1, 10, true).
		AddItem(titleInputField, 1, 0, 1, 3, 1, 10, true).
		AddItem(albumInputField, 2, 0, 1, 3, 1, 10, true).
		AddItem(getTagButton, 3, 0, 1, 3, 1, 10, true).
		AddItem(saveTagButton, 4, 0, 1, 3, 1, 10, true).
		AddItem(lyricDropDown, 6, 0, 1, 3, 1, 10, true).
		AddItem(deleteLyricButton, 7, 0, 1, 3, 1, 10, true).
		AddItem(getLyricDropDown, 9, 0, 1, 3, 1, 20, true).
		AddItem(getLyricButton, 10, 0, 1, 3, 1, 10, true)
	leftGrid.SetBorder(true).
		SetTitle(node.name).
		SetBorderPadding(1, 1, 2, 2)

	rightFlex.SetDirection(tview.FlexColumn).
		AddItem(lyricTextView, 0, 1, true)

		// lyricFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		// 	AddItem(leftGrid, 0, 2, true).
		// 	AddItem(rightFlex, 0, 3, true)

	lyricFlex := &myFlex{
		tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(leftGrid, 0, 2, true).
			AddItem(rightFlex, 0, 3, true),
		nil,
	}

	lyricFlex.
		SetTitle(node.name).
		SetBorderPadding(1, 1, 1, 1).
		SetBackgroundColor(gomu.colors.popup)

	inputs := []tview.Primitive{
		artistInputField,
		titleInputField,
		albumInputField,
		getTagButton,
		saveTagButton,
		lyricDropDown,
		deleteLyricButton,
		getLyricDropDown,
		getLyricButton,
		lyricTextView,
	}

	gomu.pages.
		AddPage(popupID, center(lyricFlex, 90, 40), true, true)
	gomu.popups.push(lyricFlex)

	lyricFlex.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyEnter:

		case tcell.KeyEsc:
			gomu.pages.RemovePage(popupID)
			gomu.popups.pop()
		case tcell.KeyTab:
			lyricFlex.cycleFocus(gomu.app, inputs, false)
		case tcell.KeyBacktab:
			lyricFlex.cycleFocus(gomu.app, inputs, true)
		case tcell.KeyRight:
			lyricFlex.cycleFocus(gomu.app, inputs, false)
		case tcell.KeyLeft:
			lyricFlex.cycleFocus(gomu.app, inputs, true)
		}

		switch e.Rune() {
		case '1':
		case '2':
		case '3':
		}
		return e
	})

	return err
}

func (f *myFlex) cycleFocus(app *tview.Application, elements []tview.Primitive, reverse bool) {
	for i, el := range elements {
		if !el.HasFocus() {
			continue
		}

		if reverse {
			i = i - 1
			if i < 0 {
				i = len(elements) - 1
			}
		} else {
			i = i + 1
			i = i % len(elements)
		}

		app.SetFocus(elements[i])
		f.FocusedItem = elements[i]
		return
	}
}

func (f *myFlex) Focus(delegate func(p tview.Primitive)) {
	// if f.FocusedItem != nil {
	// 	delegate(f.FocusedItem)
	// } else {
	// 	f.Focus(f.FocusedItem)
	// }
	// 	for _, item := range f.Focus.items {
	// 	if item.Item != nil && item.Focus {
	// 		delegate(item.Item)
	// 		return
	// 	}
	// }
	if f.FocusedItem != nil {
		gomu.app.SetFocus(f.FocusedItem)
	} else {
		f.Flex.Focus(delegate)
	}
}
func loadTagMap(node *AudioFile) (tag *id3v2.Tag, popupLyricMap map[string]string, options []string, err error) {

	popupLyricMap = make(map[string]string)

	if node.isAudioFile {
		tag, err = id3v2.Open(node.path, id3v2.Options{Parse: true})
		if err != nil {
			return nil, nil, nil, tracerr.Wrap(err)
		}
		defer tag.Close()
	} else {
		return nil, nil, nil, fmt.Errorf("not an audio file")
	}
	usltFrames := tag.GetFrames(tag.CommonID("Unsynchronised lyrics/text transcription"))

	for _, f := range usltFrames {
		uslf, ok := f.(id3v2.UnsynchronisedLyricsFrame)
		if !ok {
			die(errors.New("USLT error"))
		}
		res := uslf.Lyrics
		popupLyricMap[uslf.ContentDescriptor] = res
	}
	for option := range popupLyricMap {
		options = append(options, option)
	}
	sort.Strings(options)

	return tag, popupLyricMap, options, err
}
