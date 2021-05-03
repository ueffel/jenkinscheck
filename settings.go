package main

import (
	"encoding/json"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/lxn/walk"
	"github.com/lxn/walk/declarative"
)

type settingsWindow struct {
	*walk.Dialog
	okPB         *walk.PushButton
	cancelPB     *walk.PushButton
	URLBox       *walk.LineEdit
	URLLb        *walk.ListBox
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
	defer handlePanic()
	settings := walk.App().Settings()

	dlg := new(settingsWindow)
	urls := new(urlsModel)
	remote := new(listModel)
	own := new(listModel)

	jenkinsURLs := getJobsURLs()
	urls.items = jenkinsURLs
	var jenkinsURL string
	if len(jenkinsURLs) > 0 {
		jenkinsURL = jenkinsURLs[0]
	}
	browser, ok := settings.Get("Browser")
	if !ok {
		browser = ""
	}
	ssBuilds := getSuccessiveSuccessful()
	interval := getInterval()

	err := declarative.Dialog{
		AssignTo:      &dlg.Dialog,
		Title:         "Settings",
		Icon:          mw.Icon(),
		DefaultButton: &dlg.okPB,
		CancelButton:  &dlg.cancelPB,
		Layout:        declarative.VBox{},
		Children: []declarative.Widget{
			declarative.Composite{
				Layout: declarative.Grid{Columns: 3},
				Children: []declarative.Widget{
					declarative.Label{Text: "Add URL to Jenkins View:"},
					declarative.LineEdit{
						Alignment: declarative.AlignHCenterVCenter,
						AssignTo:  &dlg.URLBox,
						Text:      jenkinsURL,
					},
					declarative.PushButton{
						Text:        "+",
						ToolTipText: "Add the above URL to the list",
						OnClicked: func() {
							newUrls := urls.items
							url := dlg.URLBox.Text()
							if !contains(newUrls, url) {
								urls.items = append(newUrls, url)
								dlg.saveUrls()
								urls.PublishItemsReset()
							}
						},
					},
					declarative.Label{Text: "Current Jenkins Views:"},
					declarative.ListBox{
						AssignTo:       &dlg.URLLb,
						MultiSelection: true,
						Model:          urls,
						OnItemActivated: func() {
							urls.items = deleteFromStringArray(urls.items, dlg.URLLb.CurrentIndex())
							dlg.saveUrls()
							urls.PublishItemsReset()
						},
					},
					declarative.Composite{
						Layout: declarative.VBox{},
						Children: []declarative.Widget{
							declarative.PushButton{
								Text:        "x",
								ToolTipText: "Remove selected Jenkins views",
								OnClicked: func() {
									urls.items = deleteFromStringArray(urls.items, dlg.URLLb.SelectedIndexes()...)
									dlg.saveUrls()
									urls.PublishItemsReset()
								},
							},
							declarative.PushButton{
								AssignTo:    &dlg.reloadPB,
								Text:        "⟳",
								ToolTipText: "Reload all jobs",
								OnClicked: func() {
									dlg.reloadPB.SetEnabled(false)
									go func() {
										defer handlePanic()
										jobs := getJobsFromMultiple(urls.items)
										allItems := make([]*job, len(jobs.Jobs))
										for i := 0; i < len(jobs.Jobs); i++ {
											job := jobs.Jobs[i]
											allItems[i] = job
										}
										dlg.Synchronize(func() {
											defer handlePanic()
											dlg.allItems = allItems
											remote.items = substractAndFilterArray(
												dlg.allItems,
												dlg.ownItems,
												dlg.remoteFilter.Text())
											remote.PublishItemsReset()
											dlg.reloadPB.SetEnabled(true)
										})
									}()
								},
							},
						},
					},
					declarative.VSeparator{ColumnSpan: 3},
					declarative.Label{
						Text: "Browser (leave empty for default browser):",
					},
					declarative.LineEdit{
						AssignTo: &dlg.browserBox,
						Text:     browser,
						OnTextChanged: func() {
							settings.Put("Browser", dlg.browserBox.Text())
						},
					},
					declarative.PushButton{
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
					declarative.Label{Text: "Interval (in seconds):"},
					declarative.LineEdit{
						AssignTo: &dlg.intervalBox,
						Text:     strconv.Itoa(interval),
						OnTextChanged: func() {
							settings.Put("Interval", dlg.intervalBox.Text())
						},
					},
					declarative.HSpacer{},
					declarative.Label{Text: "Notify after successive successful builds:"},
					declarative.CheckBox{
						AssignTo:   &dlg.ssBox,
						Checked:    ssBuilds,
						ColumnSpan: 2,
						OnCheckedChanged: func() {
							settings.Put("Successive_successful", strconv.FormatBool(dlg.ssBox.Checked()))
						},
					},
					declarative.VSeparator{ColumnSpan: 3},
				},
			},
			declarative.Composite{
				Layout: declarative.Grid{Columns: 3},
				Children: []declarative.Widget{
					declarative.Label{
						Row:       1,
						Column:    1,
						Alignment: declarative.AlignHCenterVCenter,
						Text:      "All Jenkins Jobs",
					},
					declarative.Composite{
						Row:    2,
						Column: 1,
						Layout: declarative.HBox{},
						Children: []declarative.Widget{
							declarative.Label{Text: "Filter:"},
							declarative.LineEdit{
								AssignTo: &dlg.remoteFilter,
								OnTextChanged: func() {
									remote.items = substractAndFilterArray(
										dlg.allItems,
										dlg.ownItems,
										dlg.remoteFilter.Text())
									remote.PublishItemsReset()
								},
							},
							declarative.PushButton{
								Text:        "x",
								ToolTipText: "Empty filter box",
								MaxSize:     declarative.Size{Width: 20, Height: 10},
								OnClicked: func() {
									dlg.remoteFilter.SetText("")
								},
							},
						},
					},
					declarative.ListBox{
						AssignTo:       &dlg.remoteLb,
						Row:            3,
						Column:         1,
						MinSize:        declarative.Size{Width: 400, Height: 400},
						Model:          remote,
						MultiSelection: true,
						OnItemActivated: func() {
							items := make([]*job, len(dlg.ownItems))
							copy(items, dlg.ownItems)
							idx := dlg.remoteLb.CurrentIndex()
							found := false
							for _, item := range dlg.ownItems {
								if item.Name == remote.items[idx].Name &&
									item.Jenkins == remote.items[idx].Jenkins {
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
					declarative.Composite{
						Row:     3,
						Column:  2,
						Layout:  declarative.VBox{},
						MinSize: declarative.Size{Width: 40, Height: 40},
						MaxSize: declarative.Size{Width: 40, Height: 40},
						Children: []declarative.Widget{
							declarative.VSpacer{},
							declarative.PushButton{
								Text: "▶",
								ToolTipText: "Add selected items from the left list (all jenkins" +
									" jobs) to the right (monitored jobs)",
								OnClicked: func() {
									items := make([]*job, len(dlg.ownItems))
									copy(items, dlg.ownItems)
									for _, idx := range dlg.remoteLb.SelectedIndexes() {
										found := false
										for _, item := range dlg.ownItems {
											if item.Name == remote.items[idx].Name &&
												item.Jenkins == remote.items[idx].Jenkins {
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
									remote.items = substractAndFilterArray(
										dlg.allItems,
										dlg.ownItems,
										dlg.remoteFilter.Text())
									remote.PublishItemsReset()
									saveJobs(dlg.ownItems)
								},
							},
							declarative.PushButton{
								Text:        "◀",
								ToolTipText: "Remove selected items from the right list (monitored jobs)",
								OnClicked: func() {
									for _, idx := range dlg.ownLb.SelectedIndexes() {
										var ownIdx int
										for ownIdx = 0; ownIdx < len(dlg.ownItems); ownIdx++ {
											if dlg.ownItems[ownIdx].Name == own.items[idx].Name &&
												dlg.ownItems[ownIdx].Jenkins == own.items[idx].Jenkins {
												break
											}
										}
										dlg.ownItems = append(dlg.ownItems[:ownIdx], dlg.ownItems[ownIdx+1:]...)
									}
									own.items = substractAndFilterArray(dlg.ownItems, []*job{}, dlg.ownFilter.Text())
									own.PublishItemsReset()
									remote.items = substractAndFilterArray(
										dlg.allItems,
										dlg.ownItems,
										dlg.remoteFilter.Text())
									remote.PublishItemsReset()
									saveJobs(dlg.ownItems)
								},
							},
							declarative.VSpacer{},
						},
					},
					declarative.Label{
						Row:       1,
						Column:    3,
						Alignment: declarative.AlignHCenterVCenter,
						Text:      "Monitored Jobs",
					},
					declarative.Composite{
						Row:    2,
						Column: 3,
						Layout: declarative.HBox{},
						Children: []declarative.Widget{
							declarative.Label{Text: "Filter:"},
							declarative.LineEdit{
								AssignTo: &dlg.ownFilter,
								OnTextChanged: func() {
									own.items = substractAndFilterArray(dlg.ownItems, []*job{}, dlg.ownFilter.Text())
									own.PublishItemsReset()
								},
							},
							declarative.PushButton{
								Text:        "x",
								ToolTipText: "Empty filter box",
								MaxSize:     declarative.Size{Width: 20, Height: 10},
								OnClicked: func() {
									dlg.ownFilter.SetText("")
								},
							},
						},
					},
					declarative.ListBox{
						AssignTo:       &dlg.ownLb,
						Row:            3,
						Column:         3,
						MinSize:        declarative.Size{Width: 400, Height: 400},
						Model:          own,
						MultiSelection: true,
						OnItemActivated: func() {
							idx := dlg.ownLb.CurrentIndex()
							var ownIdx int
							for ownIdx = 0; ownIdx < len(dlg.ownItems); ownIdx++ {
								if dlg.ownItems[ownIdx].Name == own.items[idx].Name &&
									dlg.ownItems[ownIdx].Jenkins == own.items[idx].Jenkins {
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
			declarative.VSeparator{},
			declarative.Composite{
				Layout: declarative.HBox{},
				Children: []declarative.Widget{
					declarative.HSpacer{},
					declarative.PushButton{
						AssignTo: &dlg.okPB,
						Text:     "Ok",
						OnClicked: func() {
							dlg.Close(walk.DlgCmdOK)
						},
					},
					declarative.PushButton{
						AssignTo: &dlg.cancelPB,
						Text:     "Cancel",
						OnClicked: func() {
							dlg.Close(walk.DlgCmdCancel)
						},
					},
				},
			},
		},
	}.Create(mw)
	if err != nil {
		log.Println(err)
	}

	go func() {
		defer handlePanic()
		jobs := getJobsFromMultiple(jenkinsURLs)
		allItems := make([]*job, len(jobs.Jobs))
		for i := 0; i < len(jobs.Jobs); i++ {
			job := jobs.Jobs[i]
			allItems[i] = job
		}
		ownItems, migration := loadJobs()

		// Settings migration from job names to job names + jenkins url
		if migration {
			for _, item := range ownItems {
				if item.Jenkins != "" {
					continue
				}
				for _, rItem := range allItems {
					if item.Name == rItem.Name {
						item.Jenkins = rItem.Jenkins
						break
					}
				}
			}
		}

		dlg.Synchronize(func() {
			defer handlePanic()
			dlg.ownItems = ownItems
			dlg.allItems = allItems
			own.items = substractAndFilterArray(dlg.ownItems, []*job{}, dlg.ownFilter.Text())
			remote.items = substractAndFilterArray(dlg.allItems, dlg.ownItems, dlg.remoteFilter.Text())
			remote.PublishItemsReset()
			own.PublishItemsReset()
		})
	}()

	dlgResult := dlg.Run()

	if dlgResult == walk.DlgCmdOK {
		settings.Save()
		mw.reInit()
	} else {
		settings.Load()
	}
}

func (dlg *settingsWindow) saveUrls() {
	var idx int
	model, ok := dlg.URLLb.Model().(*urlsModel)
	if !ok {
		return
	}
	settings := walk.App().Settings()
	for idx = 0; idx < len(model.items); idx++ {
		settings.Put("URL_"+strconv.Itoa(idx), model.items[idx])
	}
	ok = true
	for ; ok; idx++ {
		key := "URL_" + strconv.Itoa(idx)
		_, ok = settings.Get(key)
		if ok {
			settings.Remove("URL_" + strconv.Itoa(idx))
		}
	}
}

type urlsModel struct {
	walk.ListModelBase
	items []string
}

func (m *urlsModel) ItemCount() int {
	return len(m.items)
}

func (m *urlsModel) Value(index int) interface{} {
	if index >= m.ItemCount() {
		return "???"
	}
	return m.items[index]
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
	regexSearch := true
	regex, err := regexp.Compile("(?i)" + filter)
	if err != nil {
		regexSearch = false
	}
	for i := 0; i < len(allItems); i++ {
		skip := false
		if filter != "" {
			if regexSearch {
				if regex.MatchString(allItems[i].Name) {
					skip = false
				} else {
					skip = true
				}
			} else {
				if strings.Contains(strings.ToLower(allItems[i].Name), strings.ToLower(filter)) {
					skip = false
				} else {
					skip = true
				}
			}
		}
		for _, item := range ownItems {
			if item.Name == allItems[i].Name && item.Jenkins == allItems[i].Jenkins {
				skip = true
				break
			}
		}
		if !skip {
			remoteItems = append(remoteItems, allItems[i])
		}
	}
	sort.Slice(remoteItems, func(i, j int) bool {
		return strings.ToLower(remoteItems[i].Name) < strings.ToLower(remoteItems[j].Name)
	})
	return remoteItems
}

type saveJob struct {
	Name     string
	Instance string
}

func loadJobs() ([]*job, bool) {
	settings := walk.App().Settings()
	watchedJobsStr, ok := settings.Get("Jobs")
	if ok {
		var watchedJobs []saveJob
		err := json.Unmarshal([]byte(watchedJobsStr), &watchedJobs)
		if err == nil {
			ownItems := make([]*job, len(watchedJobs))
			for i, item := range watchedJobs {
				ownItems[i] = &job{
					Name:    item.Name,
					Jenkins: item.Instance,
				}
			}
			return substractAndFilterArray(ownItems, []*job{}, ""), false
		}

		log.Println("loadJobs:", err)
		var jobStrings []string
		err = json.Unmarshal([]byte(watchedJobsStr), &jobStrings)
		if err != nil {
			log.Println("loadJobs2:", err)
		}

		ownItems := make([]*job, len(jobStrings))
		for i, item := range jobStrings {
			ownItems[i] = &job{Name: item}
		}
		return substractAndFilterArray(ownItems, []*job{}, ""), true
	}
	return []*job{}, false
}

func saveJobs(ownItems []*job) {
	settings := walk.App().Settings()
	watchedJobs := make([]saveJob, len(ownItems))
	for i, item := range ownItems {
		watchedJobs[i] = saveJob{
			Name:     item.Name,
			Instance: item.Jenkins,
		}
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

func getJobsURLs() []string {
	settings := walk.App().Settings()

	jenkinsURL, ok := settings.Get("CC_URL")
	if ok {
		settings.Remove("CC_URL")
		settings.Put("URL_0", jenkinsURL)
		settings.Save()
		return []string{jenkinsURL}
	}

	var jenkinsURLs []string
	ok = true
	for i := 0; ok; i++ {
		jenkinsURL, ok = settings.Get("URL_" + strconv.Itoa(i))
		if ok {
			jenkinsURLs = append(jenkinsURLs, jenkinsURL)
		}
	}
	if len(jenkinsURLs) > 0 {
		return jenkinsURLs
	}

	return []string{"http://hudson.pdv.lan/", "http://hudson.pdv.lan:8090/view/All%20Flat"}
}

func contains(haystack []string, needle string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}

func deleteFromStringArray(input []string, indexes ...int) []string {
	var output []string
	lastIdx := 0
	for _, idx := range indexes {
		output = append(output, input[lastIdx:idx]...)
		lastIdx = idx + 1
	}
	output = append(output, input[lastIdx:]...)
	return output
}
