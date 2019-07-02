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

func (mw *jenkinsMainWindow) openSettings() {
	settings := walk.App().Settings()
	var dlg *walk.Dialog
	var okPB *walk.PushButton
	var cancelPB *walk.PushButton
	var ccUrlBox *walk.LineEdit
	var ssBox *walk.CheckBox
	var browserBox *walk.LineEdit
	var intervalBox *walk.LineEdit
	var ownFilter *walk.LineEdit
	var remoteFilter *walk.LineEdit
	var remoteLb *walk.ListBox
	var ownLb *walk.ListBox
	remote := new(listModel)
	own := new(listModel)

	ccUrl := getCCXmlUrl()
	browser, ok := settings.Get("Browser")
	if !ok {
		browser = ""
	}
	ssBuilds := getSuccessiveSuccessful()
	interval := getInterval()
	jobs := getCCXmlJobs(ccUrl)
	allItems := make([]*job, len(jobs.Jobs))
	for i := 0; i < len(jobs.Jobs); i++ {
		job := jobs.Jobs[i]
		allItems[i] = &job
	}
	remote.items = allItems
	ownItems := loadJobs()
	own.items = ownItems
	remote.items = substractAndFilterArray(allItems, ownItems, "")
	remote.PublishItemsReset()

	dlgResult, err := Dialog{
		AssignTo:      &dlg,
		Title:         "Settings",
		Icon:          mw.Icon(),
		DefaultButton: &okPB,
		CancelButton:  &cancelPB,
		Layout:        VBox{},
		Children: []Widget{
			Composite{
				MaxSize: Size{800, 500},
				Layout:  Grid{Columns: 3},
				Children: []Widget{
					Label{Text: "URL to cc.xml:"},
					LineEdit{
						Alignment: AlignHCenterVCenter,
						AssignTo:  &ccUrlBox,
						Text:      ccUrl,
						OnTextChanged: func() {
							settings.Put("CC_URL", ccUrlBox.Text())
						},
					},
					PushButton{
						Text: "⟳",
						OnClicked: func() {
							jobs = getCCXmlJobs(ccUrlBox.Text())
							allItems = make([]*job, len(jobs.Jobs))
							for i := 0; i < len(jobs.Jobs); i++ {
								job := jobs.Jobs[i]
								allItems[i] = &job
							}
							remote.items = substractAndFilterArray(allItems, ownItems, remoteFilter.Text())
							remote.PublishItemsReset()
						},
					},
					Label{
						Text: "Browser (leave empty for default browser):",
					},
					LineEdit{
						AssignTo: &browserBox,
						Text:     browser,
						OnTextChanged: func() {
							settings.Put("Browser", browserBox.Text())
						},
					},
					PushButton{
						Text: "Browse",
						OnClicked: func() {
							dlg := new(walk.FileDialog)
							dlg.Filter = "Executables (*.exe)|*.exe"
							ok, err := dlg.ShowOpen(mw)
							if err != nil {
								log.Println(err)
							}
							if !ok {
								return
							}
							browserBox.SetText(dlg.FilePath)
						},
					},
					Label{Text: "Interval (in seconds):"},
					LineEdit{
						AssignTo: &intervalBox,
						Text:     strconv.Itoa(interval),
						OnTextChanged: func() {
							settings.Put("Interval", intervalBox.Text())
						},
					},
					HSpacer{},
					CheckBox{
						AssignTo:   &ssBox,
						Text:       "Notify after successive successful builds",
						Checked:    bool(ssBuilds),
						ColumnSpan: 3,
						OnCheckedChanged: func() {
							settings.Put("Successive_successful", strconv.FormatBool(ssBox.Checked()))
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
										AssignTo: &remoteFilter,
										OnTextChanged: func() {
											remote.items = substractAndFilterArray(allItems, ownItems, remoteFilter.Text())
											remote.PublishItemsReset()
										},
									},
									PushButton{
										Text:    "x",
										MaxSize: Size{20, 10},
										OnClicked: func() {
											remoteFilter.SetText("")
										},
									},
								},
							},
							ListBox{
								AssignTo:       &remoteLb,
								MinSize:        Size{400, 400},
								Model:          remote,
								MultiSelection: true,
								OnItemActivated: func() {
									items := make([]*job, len(ownItems))
									copy(items, ownItems)
									idx := remoteLb.CurrentIndex()
									found := false
									for _, item := range ownItems {
										if item.Name == remote.items[idx].Name {
											found = true
											break
										}
									}
									if !found {
										items = append(items, remote.items[idx])
									}
									ownItems = items
									own.items = substractAndFilterArray(ownItems, []*job{}, ownFilter.Text())
									own.PublishItemsReset()
									remote.items = substractAndFilterArray(allItems, ownItems, remoteFilter.Text())
									remote.PublishItemsReset()
									saveJobs(ownItems)
								},
							},
						},
					},
					Composite{
						Layout:  VBox{},
						MinSize: Size{40, 40},
						MaxSize: Size{40, 40},
						Children: []Widget{
							HSpacer{},
							PushButton{
								Text: "▶",
								OnClicked: func() {
									items := make([]*job, len(ownItems))
									copy(items, ownItems)
									for _, idx := range remoteLb.SelectedIndexes() {
										found := false
										for _, item := range ownItems {
											if item.Name == remote.items[idx].Name {
												found = true
												break
											}
										}
										if !found {
											items = append(items, remote.items[idx])
										}
									}
									ownItems = items
									own.items = substractAndFilterArray(ownItems, []*job{}, ownFilter.Text())
									own.PublishItemsReset()
									remote.items = substractAndFilterArray(allItems, ownItems, remoteFilter.Text())
									remote.PublishItemsReset()
									saveJobs(ownItems)
								},
							},
							PushButton{
								Text: "◀",
								OnClicked: func() {
									items := []*job{}
									log.Println(ownLb.SelectedIndexes())
									lastIdx := 0
									for _, idx := range ownLb.SelectedIndexes() {
										var ownIdx int
										for ownIdx = 0; ownIdx < len(ownItems); ownIdx++ {
											if ownItems[ownIdx].Name == own.items[idx].Name {
												break
											}
										}
										ownItems = append(ownItems[:ownIdx], ownItems[ownIdx+1:]...)
										items = append(items, own.items[lastIdx:idx]...)
										lastIdx = idx + 1
									}
									items = append(items, own.items[lastIdx:]...)
									own.items = substractAndFilterArray(ownItems, []*job{}, ownFilter.Text())
									own.PublishItemsReset()
									remote.items = substractAndFilterArray(allItems, ownItems, remoteFilter.Text())
									remote.PublishItemsReset()
									saveJobs(ownItems)
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
										AssignTo: &ownFilter,
										OnTextChanged: func() {
											own.items = substractAndFilterArray(ownItems, []*job{}, ownFilter.Text())
											own.PublishItemsReset()
										},
									},
									PushButton{
										Text:    "x",
										MaxSize: Size{20, 10},
										OnClicked: func() {
											ownFilter.SetText("")
										},
									},
								},
							},
							ListBox{
								AssignTo:       &ownLb,
								MinSize:        Size{400, 400},
								Model:          own,
								MultiSelection: true,
								OnItemActivated: func() {
									items := []*job{}
									idx := ownLb.CurrentIndex()
									items = append(items, own.items[:idx]...)
									items = append(items, own.items[idx+1:]...)
									var ownIdx int
									for ownIdx = 0; ownIdx < len(ownItems); ownIdx++ {
										if ownItems[ownIdx].Name == own.items[idx].Name {
											break
										}
									}
									ownItems = append(ownItems[:ownIdx], ownItems[ownIdx+1:]...)
									own.items = substractAndFilterArray(ownItems, []*job{}, ownFilter.Text())
									own.PublishItemsReset()
									remote.items = substractAndFilterArray(allItems, ownItems, remoteFilter.Text())
									remote.PublishItemsReset()
									saveJobs(ownItems)
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
						AssignTo: &okPB,
						Text:     "Ok",
						OnClicked: func() {
							dlg.Close(walk.DlgCmdOK)
						},
					},
					PushButton{
						AssignTo: &cancelPB,
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
	} else {
		return []*job{}
	}
}

func saveJobs(ownItems []*job) {
	settings := walk.App().Settings()
	watchedJobs := make([]string, len(ownItems))
	for i, item := range ownItems {
		watchedJobs[i] = item.Name
	}
	watchedJobsJson, _ := json.Marshal(watchedJobs)
	log.Println(string(watchedJobsJson))
	settings.Put("Jobs", string(watchedJobsJson))
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

func getCCXmlUrl() string {
	settings := walk.App().Settings()
	ccUrl, ok := settings.Get("CC_URL")
	if !ok {
		ccUrl = "http://hudson.pdv.lan/cc.xml"
	}
	return ccUrl
}
