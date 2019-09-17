package main

import (
	"encoding/json"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type settingsWindow struct {
	*walk.Dialog
	okPB         *walk.PushButton
	cancelPB     *walk.PushButton
	ccURLBox     *walk.LineEdit
	ssBox        *walk.CheckBox
	reloadPB     *walk.PushButton
	browserBox   *walk.LineEdit
	intervalBox  *walk.LineEdit
	ownFilter    *walk.LineEdit
	remoteFilter *walk.LineEdit
	remoteLb     *walk.ListBox
	ownLb        *walk.ListBox
	allItems     []*job
	ownItems     []*job
}

func (mw *jenkinsMainWindow) openSettings() {
	settings := walk.App().Settings()

	dlg := new(settingsWindow)
	remote := new(listModel)
	own := new(listModel)

	ccURL := getCCXmlURL()
	browser, ok := settings.Get("Browser")
	if !ok {
		browser = ""
	}
	ssBuilds := getSuccessiveSuccessful()
	interval := getInterval()
	go func() {
		jobs := getCCXmlJobs(ccURL)
		dlg.allItems = make([]*job, len(jobs.Jobs))
		for i := 0; i < len(jobs.Jobs); i++ {
			job := jobs.Jobs[i]
			dlg.allItems[i] = &job
		}
		remote.items = dlg.allItems
		dlg.ownItems = loadJobs()
		own.items = dlg.ownItems
		remote.items = substractAndFilterArray(dlg.allItems, dlg.ownItems, "")
		dlg.Synchronize(func() {
			remote.PublishItemsReset()
			own.PublishItemsReset()
		})
	}()

	dlgResult, err := Dialog{
		AssignTo:      &dlg.Dialog,
		Title:         "Settings",
		Icon:          mw.Icon(),
		DefaultButton: &dlg.okPB,
		CancelButton:  &dlg.cancelPB,
		Layout:        VBox{},
		Children: []Widget{
			Composite{
				MaxSize: Size{Width: 800, Height: 500},
				Layout:  Grid{Columns: 3},
				Children: []Widget{
					Label{Text: "URL to cc.xml:"},
					LineEdit{
						Alignment: AlignHCenterVCenter,
						AssignTo:  &dlg.ccURLBox,
						Text:      ccURL,
						OnTextChanged: func() {
							settings.Put("CC_URL", dlg.ccURLBox.Text())
						},
					},
					PushButton{
						AssignTo: &dlg.reloadPB,
						Text:     "⟳",
						OnClicked: func() {
							dlg.reloadPB.SetEnabled(false)
							go func() {
								jobs := getCCXmlJobs(dlg.ccURLBox.Text())
								dlg.allItems = make([]*job, len(jobs.Jobs))
								for i := 0; i < len(jobs.Jobs); i++ {
									job := jobs.Jobs[i]
									dlg.allItems[i] = &job
								}
								remote.items = substractAndFilterArray(dlg.allItems, dlg.ownItems, dlg.remoteFilter.Text())
								dlg.Synchronize(func() {
									remote.PublishItemsReset()
									dlg.reloadPB.SetEnabled(true)
								})
							}()
						},
					},
					Label{
						Text: "Browser (leave empty for default browser):",
					},
					LineEdit{
						AssignTo: &dlg.browserBox,
						Text:     browser,
						OnTextChanged: func() {
							settings.Put("Browser", dlg.browserBox.Text())
						},
					},
					PushButton{
						Text: "Browse",
						OnClicked: func() {
							fileDlg := new(walk.FileDialog)
							fileDlg.Filter = "Executables (*.exe)|*.exe"
							ok, err := fileDlg.ShowOpen(mw)
							if err != nil {
								log.Println(err)
							}
							if !ok {
								return
							}
							dlg.browserBox.SetText(fileDlg.FilePath)
						},
					},
					Label{Text: "Interval (in seconds):"},
					LineEdit{
						AssignTo: &dlg.intervalBox,
						Text:     strconv.Itoa(interval),
						OnTextChanged: func() {
							settings.Put("Interval", dlg.intervalBox.Text())
						},
					},
					HSpacer{},
					CheckBox{
						AssignTo:   &dlg.ssBox,
						Text:       "Notify after successive successful builds",
						Checked:    bool(ssBuilds),
						ColumnSpan: 3,
						OnCheckedChanged: func() {
							settings.Put("Successive_successful", strconv.FormatBool(dlg.ssBox.Checked()))
						},
					},
				},
			},
			VSeparator{},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Composite{
						Layout: VBox{},
						Children: []Widget{
							Composite{
								Layout: HBox{},
								Children: []Widget{
									Label{Text: "Filter:"},
									LineEdit{
										AssignTo: &dlg.remoteFilter,
										OnTextChanged: func() {
											remote.items = substractAndFilterArray(dlg.allItems, dlg.ownItems, dlg.remoteFilter.Text())
											remote.PublishItemsReset()
										},
									},
									PushButton{
										Text:    "x",
										MaxSize: Size{Width: 20, Height: 10},
										OnClicked: func() {
											dlg.remoteFilter.SetText("")
										},
									},
								},
							},
							ListBox{
								AssignTo:       &dlg.remoteLb,
								MinSize:        Size{Width: 400, Height: 400},
								Model:          remote,
								MultiSelection: true,
								OnItemActivated: func() {
									items := make([]*job, len(dlg.ownItems))
									copy(items, dlg.ownItems)
									idx := dlg.remoteLb.CurrentIndex()
									found := false
									for _, item := range dlg.ownItems {
										if item.Name == remote.items[idx].Name {
											found = true
											break
										}
									}
									if !found {
										items = append(items, remote.items[idx])
									}
									dlg.ownItems = items
									own.items = substractAndFilterArray(dlg.ownItems, []*job{}, dlg.ownFilter.Text())
									own.PublishItemsReset()
									remote.items = substractAndFilterArray(dlg.allItems, dlg.ownItems, dlg.remoteFilter.Text())
									remote.PublishItemsReset()
									saveJobs(dlg.ownItems)
								},
							},
						},
					},
					Composite{
						Layout:  VBox{},
						MinSize: Size{Width: 40, Height: 40},
						MaxSize: Size{Width: 40, Height: 40},
						Children: []Widget{
							HSpacer{},
							PushButton{
								Text: "▶",
								OnClicked: func() {
									items := make([]*job, len(dlg.ownItems))
									copy(items, dlg.ownItems)
									for _, idx := range dlg.remoteLb.SelectedIndexes() {
										found := false
										for _, item := range dlg.ownItems {
											if item.Name == remote.items[idx].Name {
												found = true
												break
											}
										}
										if !found {
											items = append(items, remote.items[idx])
										}
									}
									dlg.ownItems = items
									own.items = substractAndFilterArray(dlg.ownItems, []*job{}, dlg.ownFilter.Text())
									own.PublishItemsReset()
									remote.items = substractAndFilterArray(dlg.allItems, dlg.ownItems, dlg.remoteFilter.Text())
									remote.PublishItemsReset()
									saveJobs(dlg.ownItems)
								},
							},
							PushButton{
								Text: "◀",
								OnClicked: func() {
									items := []*job{}
									lastIdx := 0
									for _, idx := range dlg.ownLb.SelectedIndexes() {
										var ownIdx int
										for ownIdx = 0; ownIdx < len(dlg.ownItems); ownIdx++ {
											if dlg.ownItems[ownIdx].Name == own.items[idx].Name {
												break
											}
										}
										dlg.ownItems = append(dlg.ownItems[:ownIdx], dlg.ownItems[ownIdx+1:]...)
										items = append(items, own.items[lastIdx:idx]...)
										lastIdx = idx + 1
									}
									items = append(items, own.items[lastIdx:]...)
									own.items = substractAndFilterArray(dlg.ownItems, []*job{}, dlg.ownFilter.Text())
									own.PublishItemsReset()
									remote.items = substractAndFilterArray(dlg.allItems, dlg.ownItems, dlg.remoteFilter.Text())
									remote.PublishItemsReset()
									saveJobs(dlg.ownItems)
								},
							},
							HSpacer{},
						},
					},
					Composite{
						Layout: VBox{},
						Children: []Widget{
							Composite{
								Layout: HBox{},
								Children: []Widget{
									Label{Text: "Filter:"},
									LineEdit{
										AssignTo: &dlg.ownFilter,
										OnTextChanged: func() {
											own.items = substractAndFilterArray(dlg.ownItems, []*job{}, dlg.ownFilter.Text())
											own.PublishItemsReset()
										},
									},
									PushButton{
										Text:    "x",
										MaxSize: Size{Width: 20, Height: 10},
										OnClicked: func() {
											dlg.ownFilter.SetText("")
										},
									},
								},
							},
							ListBox{
								AssignTo:       &dlg.ownLb,
								MinSize:        Size{Width: 400, Height: 400},
								Model:          own,
								MultiSelection: true,
								OnItemActivated: func() {
									items := []*job{}
									idx := dlg.ownLb.CurrentIndex()
									items = append(items, own.items[:idx]...)
									items = append(items, own.items[idx+1:]...)
									var ownIdx int
									for ownIdx = 0; ownIdx < len(dlg.ownItems); ownIdx++ {
										if dlg.ownItems[ownIdx].Name == own.items[idx].Name {
											break
										}
									}
									dlg.ownItems = append(dlg.ownItems[:ownIdx], dlg.ownItems[ownIdx+1:]...)
									own.items = substractAndFilterArray(dlg.ownItems, []*job{}, dlg.ownFilter.Text())
									own.PublishItemsReset()
									remote.items = substractAndFilterArray(dlg.allItems, dlg.ownItems, dlg.remoteFilter.Text())
									remote.PublishItemsReset()
									saveJobs(dlg.ownItems)
								},
							},
						},
					},
				},
			},
			VSeparator{},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &dlg.okPB,
						Text:     "Ok",
						OnClicked: func() {
							dlg.Close(walk.DlgCmdOK)
						},
					},
					PushButton{
						AssignTo: &dlg.cancelPB,
						Text:     "Cancel",
						OnClicked: func() {
							dlg.Close(walk.DlgCmdCancel)
						},
					},
				},
			},
		},
	}.Run(mw)

	if err != nil {
		log.Println(err)
	}
	if dlgResult == walk.DlgCmdOK {
		settings.Save()
		mw.reInit()
	} else {
		settings.Load()
	}
}

type listModel struct {
	walk.ListModelBase
	items []*job
}

func (m *listModel) ItemCount() int {
	return len(m.items)
}

func (m *listModel) Value(index int) interface{} {
	if index >= m.ItemCount() {
		return "???"
	}
	return m.items[index].Name
}

func substractAndFilterArray(allItems []*job, ownItems []*job, filter string) []*job {
	remoteItems := []*job{}
	for i := 0; i < len(allItems); i++ {
		skip := false
		if filter != "" {
			if strings.Contains(strings.ToLower(allItems[i].Name), strings.ToLower(filter)) {
				skip = false
			} else {
				skip = true
			}
		}
		for _, item := range ownItems {
			if item.Name == allItems[i].Name {
				skip = true
				break
			}
		}
		if !skip {
			remoteItems = append(remoteItems, allItems[i])
		}
	}
	sort.Slice(remoteItems, func(i, j int) bool {
		return remoteItems[i].Name < remoteItems[j].Name
	})
	return remoteItems
}

func loadJobs() []*job {
	settings := walk.App().Settings()
	watchedJobsStr, ok := settings.Get("Jobs")
	if ok {
		var watchedJobs []string
		json.Unmarshal([]byte(watchedJobsStr), &watchedJobs)
		ownItems := make([]*job, len(watchedJobs))
		for i, item := range watchedJobs {
			ownItems[i] = &job{Name: item}
		}
		return substractAndFilterArray(ownItems, []*job{}, "")
	}
	return []*job{}
}

func saveJobs(ownItems []*job) {
	settings := walk.App().Settings()
	watchedJobs := make([]string, len(ownItems))
	for i, item := range ownItems {
		watchedJobs[i] = item.Name
	}
	watchedJobsJSON, _ := json.Marshal(watchedJobs)
	settings.Put("Jobs", string(watchedJobsJSON))
}

func getInterval() int {
	settings := walk.App().Settings()
	var interval int
	intervalStr, ok := settings.Get("Interval")
	if !ok {
		interval = 30
	} else {
		interval, _ = strconv.Atoi(intervalStr)
	}
	return interval
}

func getSuccessiveSuccessful() bool {
	settings := walk.App().Settings()
	ssBuildsStr, ok := settings.Get("Successive_successful")
	var ssBuilds bool
	if !ok {
		ssBuilds = false
	} else {
		ssBuilds, _ = strconv.ParseBool(ssBuildsStr)
	}
	return ssBuilds
}

func getCCXmlURL() string {
	settings := walk.App().Settings()
	ccURL, ok := settings.Get("CC_URL")
	if !ok {
		ccURL = "http://hudson.pdv.lan/cc.xml"
	}
	return ccURL
}
