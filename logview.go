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
	txt *walk.TextEdit
}

func (mw *jenkinsMainWindow) openLogView(j *job) {
	lv := new(logview)

	err := (Dialog{
		AssignTo: &lv.Dialog,
		Title:    "Console Log",
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
				MaxLength:       1000000,
				ReadOnly:        true,
				DoubleBuffering: true,

				Font: Font{
					Family:    "Consolas",
					PointSize: 12,
				},
			},
		},
	}).Create(mw)
	if err != nil {
		log.Println(err)
		return
	}

	go func() {
		lv.SetText("Getting build log...")
		resp, err := http.Get(fmt.Sprint(j.URL, "/", j.LastBuild.Label, "/consoleText"))
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
		reader := bufio.NewReader(resp.Body)
		var builder strings.Builder
		lv.SetText("")

		ticker := time.NewTicker(200 * time.Millisecond)
		stopUpdating := make(chan bool)
		defer close(stopUpdating)
		go func(b *strings.Builder) {
			for {
				select {
				case <-stopUpdating:
					ticker.Stop()
					lv.AppendText(b.String())
					return
				case <-ticker.C:
					lv.AppendText(b.String())
					b.Reset()
				}
			}
		}(&builder)

		for {
			line, err := reader.ReadString('\n')
			if errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				lv.AppendText(fmt.Sprintln("Reading Response failed:", err))
				return
			}
			if len(line) > 0 {

				builder.WriteString(line[:len(line)-1])
				builder.WriteString("\r\n")
			}
		}
		stopUpdating <- true
	}()
	lv.Run()
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
		}

	})
}
