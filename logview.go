package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/chzyer/readline/runes"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

type logview struct {
	*walk.Dialog
	txt       *walk.TextEdit
	searchBox *walk.LineEdit
	searchPB  *walk.PushButton
	prevPB    *walk.PushButton
	nextPB    *walk.PushButton
	closePB   *walk.PushButton
	refreshPB *walk.PushButton
	job       *job
	searchPos int
}

func (mw *jenkinsMainWindow) openLogView(j *job) {
	defer handlePanic()
	lv := &logview{
		job:       j,
		searchPos: 0,
	}

	err := (Dialog{
		AssignTo:      &lv.Dialog,
		Title:         "Console Log",
		DefaultButton: &lv.searchPB,
		CancelButton:  &lv.closePB,
		MinSize: Size{
			Height: 800,
			Width:  1200,
		},
		Layout: VBox{MarginsZero: true},
		Children: []Widget{
			TextEdit{
				AssignTo:        &lv.txt,
				HScroll:         true,
				VScroll:         true,
				MaxLength:       500000,
				ReadOnly:        true,
				DoubleBuffering: true,
				Font: Font{
					Family:    "Consolas",
					PointSize: 12,
				},
				OnKeyPress: func(key walk.Key) {
					if key == walk.KeyF3 {
						lv.searchText(!walk.ShiftDown())
					}
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					LineEdit{
						AssignTo:    &lv.searchBox,
						ToolTipText: "Search",
						MaxSize:     Size{Width: 300},
					},
					PushButton{
						AssignTo:    &lv.searchPB,
						Text:        "🔍",
						ToolTipText: "Search from the beginning (Enter)",
						MaxSize:     Size{Width: 20},
						OnClicked: func() {
							lv.searchPos = -1
							lv.searchText(true)
						},
					},
					PushButton{
						AssignTo:    &lv.prevPB,
						Text:        "<",
						ToolTipText: "Previous match (Shift+F3)",
						MaxSize:     Size{Width: 20},
						OnClicked: func() {
							lv.searchText(false)
						},
					},
					PushButton{
						AssignTo:    &lv.nextPB,
						Text:        ">",
						ToolTipText: "Next match (F3)",
						MaxSize:     Size{Width: 20},
						OnClicked: func() {
							lv.searchText(true)
						},
					},
					HSpacer{},
					PushButton{
						AssignTo:    &lv.refreshPB,
						Text:        "Refresh",
						ToolTipText: "Redownload the console log",
						OnClicked: func() {
							go lv.LoadText()
						},
					},
					PushButton{
						AssignTo:    &lv.closePB,
						Text:        "Ok",
						ToolTipText: "Close the log window (ESC)",
						OnClicked: func() {
							lv.Close(walk.DlgCmdOK)
						},
					},
				},
			},
		},
	}).Create(mw)
	if err != nil {
		log.Println(err)
		return
	}

	go lv.LoadText()
	lv.Run()
}

func (lv *logview) LoadText() {
	defer handlePanic()
	lv.SetText("Getting build log...")
	resp, err := http.Get(fmt.Sprint(lv.job.URL, "/", lv.job.LastBuild.Label, "/consoleText"))
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		lv.AppendText(fmt.Sprintln("Log Request failed:", err))
		return
	}
	if resp.StatusCode != http.StatusOK {
		lv.AppendText(fmt.Sprintln("Log Request not OK:", resp.StatusCode, resp.Status))
		return
	}
	timeout := time.AfterFunc(5*time.Second, func() {
		resp.Body.Close()
	})
	reader := bufio.NewReader(resp.Body)
	lv.SetText("")

	stopUpdating := make(chan bool)
	defer close(stopUpdating)
	textChan := make(chan string, 50)
	defer close(textChan)
	go func(txt <-chan string) {
		defer handlePanic()
		ticker := time.NewTicker(200 * time.Millisecond)
		var builder strings.Builder
		var l string
		var txtOpen bool
		for {
			select {
			case l, txtOpen = <-txt:
				if txtOpen && len(l) > 0 {
					if strings.HasSuffix(l, "\r\n") {
						builder.WriteString(l)
					} else {
						builder.WriteString(l[:len(l)-1])
						builder.WriteString("\r\n")
					}
				}
			case <-stopUpdating:
				// wait until all text is printed before exiting
				if txtOpen || builder.Len() > 0 {
					continue
				}
				ticker.Stop()
				return
			case <-ticker.C:
				if builder.Len() > 0 {
					lv.AppendText(builder.String())
					builder.Reset()
				}
			}
		}
	}(textChan)

	for {
		timeout.Reset(2 * time.Second)
		line, err := reader.ReadString('\n')
		if errors.Is(err, io.EOF) {
			if len(line) == 0 {
				break
			}
		} else if err != nil {
			lv.AppendText(fmt.Sprintln("Reading Response failed:", err))
			break
		}
		textChan <- line
	}
	stopUpdating <- true
}

func (lv *logview) SetText(txt string) {
	lv.Synchronize(func() {
		defer handlePanic()
		newLen := len(txt)
		if newLen > lv.txt.MaxLength() {
			lv.txt.SetText(txt[newLen-lv.txt.MaxLength():])
		} else {
			lv.txt.SetText(txt)
		}
	})
}

func (lv *logview) AppendText(txt string) {
	lv.Synchronize(func() {
		defer handlePanic()
		newLen := len(txt)
		// zero bytes are evil
		txt = strings.ReplaceAll(txt, "\x00", " ")
		if lv.txt.TextLength()+newLen >= lv.txt.MaxLength() {
			if newLen > lv.txt.MaxLength() {
				lv.txt.SetText("")
				lv.txt.AppendText(txt[newLen-lv.txt.MaxLength():])
			} else {
				lv.txt.SetText(lv.txt.Text()[newLen:])
				lv.txt.AppendText(txt)
			}
		} else {
			lv.txt.AppendText(txt)
		}
	})
}

func (lv *logview) searchText(forward bool) {
	searchTerm := []rune(strings.ToLower(lv.searchBox.Text()))
	if len(searchTerm) == 0 {
		return
	}
	haystack := []rune(strings.ToLower(lv.txt.Text()))
	var startSearch int
	if lv.searchPos == -1 {
		startSearch = 0
	} else {
		startSearch, _ = lv.txt.TextSelection()
		startSearch++
	}
	var startSelection int
	if forward {
		startSelection = runes.IndexAll(haystack[startSearch:], searchTerm)
		if startSelection != -1 {
			startSelection += startSearch
		}
	} else {
		if startSearch <= 0 {
			startSearch = len(haystack) - 1
		}
		startSelection = runes.IndexAllBck(haystack[:startSearch], searchTerm)
	}
	lv.Synchronize(func() {
		defer handlePanic()
		lv.searchPos = startSelection
		if startSelection == -1 {
			win.MessageBeep(win.MB_ICONEXCLAMATION)
			return
		}
		lv.txt.SetTextSelection(startSelection, startSelection+len(searchTerm))
		lv.txt.ScrollToCaret()
		lv.txt.SetFocus()
	})
}
