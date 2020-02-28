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

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type logview struct {
	*walk.Dialog
	txt      *walk.TextEdit
	okPB     *walk.PushButton
	cancelPB *walk.PushButton
	job      *job
}

func (mw *jenkinsMainWindow) openLogView(j *job) {
	lv := &logview{
		job: j,
	}

	err := (Dialog{
		AssignTo:      &lv.Dialog,
		Title:         "Console Log",
		DefaultButton: &lv.okPB,
		CancelButton:  &lv.okPB,
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
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &lv.cancelPB,
						Text:     "Refresh",
						OnClicked: func() {
							go lv.LoadText()
						},
					},
					PushButton{
						AssignTo: &lv.okPB,
						Text:     "Ok",
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

	ticker := time.NewTicker(200 * time.Millisecond)
	stopUpdating := make(chan bool)
	defer close(stopUpdating)
	textChan := make(chan string, 10)
	defer close(textChan)
	go func(txt <-chan string) {
		var builder strings.Builder
		for {
			select {
			case l := <-txt:
				if len(l) > 0 {
					if strings.HasSuffix(l, "\r\n") {
						builder.WriteString(l)
					} else {
						builder.WriteString(l[:len(l)-1])
						builder.WriteString("\r\n")
					}
				}
			case <-stopUpdating:
				ticker.Stop()
				lv.AppendText(builder.String())
				return
			case <-ticker.C:
				lv.AppendText(builder.String())
				builder.Reset()
			}
		}
	}(textChan)

	for {
		timeout.Reset(1 * time.Second)
		line, err := reader.ReadString('\n')
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			lv.AppendText(fmt.Sprintln("Reading Response failed:", err))
			return
		}
		textChan <- line
	}
	stopUpdating <- true
}

func (lv *logview) SetText(txt string) {
	lv.Synchronize(func() {
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
		newLen := len(txt)
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
