//go:generate goversioninfo -o rsrc.syso
// JenkinsCheck project main.go
package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

func main() {
	appName := "JenkinsCheck"
	company := "PDV GmbH"
	localAppData, _ := walk.LocalAppDataPath()
	logDir := path.Join(localAppData, company, appName)
	err := os.MkdirAll(logDir, 0644)
	if err != nil {
		log.Fatalf("error creating directories: %v", err)
	}
	logPath := path.Join(logDir, "JenkinsCheck.log")
	logger, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer logger.Close()
	multiLogger := io.MultiWriter(os.Stdout, logger)
	log.SetOutput(multiLogger)
	proxy := http.ProxyFromEnvironment
	trans := &http.Transport{Proxy: proxy}
	http.DefaultTransport = trans
	http.DefaultClient.Timeout = 5 * time.Second

	mainWindow := new(jenkinsMainWindow)
	boldFont, _ := walk.NewFont("Calibri", 18, walk.FontBold)
	solidWhite, _ := walk.NewSolidColorBrush(walk.RGB(255, 255, 255))

	icon, err := walk.NewIconFromResourceId(2)
	if err != nil {
		log.Fatal(err)
	}
	app := walk.App()
	app.SetOrganizationName(company)
	app.SetProductName(appName)
	settings := walk.NewIniFileSettings("settings.ini")
	err = settings.Load()
	if err != nil {
		log.Fatal(err)
	}
	app.SetSettings(settings)

	tableModel := new(jobModel)

	MainWindow{
		AssignTo: &mainWindow.MainWindow,
		Name:     "MainWindow",
		Icon:     icon,
		Title:    appName,
		Size:     Size{Width: 900, Height: 800},
		Font:     Font{Family: "Calibri", PointSize: 12},
		MenuItems: []MenuItem{
			Menu{
				Text: "&Settings",
				Items: []MenuItem{
					Action{
						Text:        "Settings",
						OnTriggered: mainWindow.openSettings,
					},
					Action{
						Text:        "E&xit",
						OnTriggered: doExit,
					},
				},
			},
		},
		Layout: HBox{},
		Children: []Widget{
			TableView{
				AssignTo:         &mainWindow.table,
				Name:             "tableView",
				AlternatingRowBG: true,
				ColumnsOrderable: true,
				Columns: []TableViewColumn{
					{
						Title: "Status",
						Name:  "LastCompletedBuild.Result",
						Width: 70,
					},
					{
						Title: "Name",
						Name:  "Name",
						Width: 400,
					},
					{
						Title: "Label",
						Name:  "LastBuild.Label",
					},
					{
						Title: "Activity",
						Name:  "LastBuild.Building",
					},
					{
						Title:  "BuildTime",
						Name:   "LastBuild.Timestamp",
						Format: "2006-01-02 15:04:05",
						Width:  150,
					},
					{
						Title: "Jenkins URL",
						Name:  "Jenkins",
						Width: 200,
					},
				},
				Model:           tableModel,
				CustomRowHeight: 30,
				StyleCell: func(style *walk.CellStyle) {
					item := tableModel.items[style.Row()]

					if style.Col() < 0 {
						return
					}

					switch mainWindow.table.Columns().At(style.Col()).Title() {
					case "Status":
						style.Font = boldFont
						switch item.LastCompletedBuild.Result {
						case "SUCCESS":
							canvas := style.Canvas()
							if canvas != nil {
								canvas.GradientFillRectangle(walk.RGB(100, 200, 100), walk.RGB(200, 250, 200), walk.Horizontal, style.Bounds())
								canvas.DrawText("👍", boldFont, walk.RGB(0, 0, 0), style.Bounds(), 127)
							}
						case "UNSTABLE":
							canvas := style.Canvas()
							if canvas != nil {
								canvas.GradientFillRectangle(walk.RGB(150, 150, 0), walk.RGB(250, 250, 0), walk.Horizontal, style.Bounds())
								canvas.DrawText("👀", boldFont, walk.RGB(0, 0, 0), style.Bounds(), 127)
							}
						case "FAILURE":
							canvas := style.Canvas()
							if canvas != nil {
								canvas.GradientFillRectangle(walk.RGB(200, 0, 0), walk.RGB(100, 0, 0), walk.Horizontal, style.Bounds())
								canvas.DrawText("👎", boldFont, walk.RGB(0, 0, 0), style.Bounds(), 127)
							}
						case "":
							fallthrough
						default:
							canvas := style.Canvas()
							if canvas != nil {
								canvas.FillRectangle(solidWhite, style.Bounds())
								canvas.DrawText("🤷‍", boldFont, walk.RGB(0, 0, 0), style.Bounds(), 127)
							}
						}
					case "Activity":
						canvas := style.Canvas()
						if !item.LastBuild.Building {
							if canvas != nil {
								canvas.DrawText("💤", boldFont, walk.RGB(0, 0, 0), style.Bounds(), 127)
							}
						} else {
							if canvas != nil {
								canvas.DrawText("🔨", boldFont, walk.RGB(0, 0, 0), style.Bounds(), 127)
							}
						}
					default:
						style.Font = mainWindow.table.Font()
					}
				},
				OnItemActivated: func() {
					exe, ok := settings.Get("Browser")
					if !ok || exe == "" {
						exe = "explorer.exe" // default browser
					}
					cmd := exec.Command(exe, tableModel.items[mainWindow.table.CurrentIndex()].URL)
					cmd.Start()
				},
			},
		},
	}.Create()

	tableModel.items = []*job{{Name: "Loading..."}}
	tableModel.PublishRowsReset()

	ni, _ := walk.NewNotifyIcon(mainWindow)
	defer ni.Dispose()
	ni.SetIcon(icon)
	ni.SetToolTip("Click to show or use the context menu to exit.")
	ni.SetVisible(true)

	showMainWindow := func() {
		mainWindow.Show()
		win.SetForegroundWindow(mainWindow.Handle())
	}

	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button != walk.LeftButton {
			return
		}
		showMainWindow()
	})

	exitAction := walk.NewAction()
	exitAction.SetText("E&xit")
	exitAction.Triggered().Attach(doExit)
	ni.ContextMenu().Actions().Add(exitAction)

	go tableModel.initJobs(ni)

	interval := getInterval()

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	quit := make(chan bool, 1)
	go func() {
		for {
			select {
			case <-ticker.C:
				tableModel.updateJobs(ni)
			case <-quit:
				ticker.Stop()
				break
			}
		}
	}()
	defer close(quit)

	walk.InitWrapperWindow(mainWindow)
	mainWindow.Run()
	quit <- true
	log.Println("saving settings")
	if err := settings.Save(); err != nil {
		log.Fatal(err)
	}
	log.Println("closing")
}

type jobModel struct {
	walk.SortedReflectTableModelBase
	items []*job
}

func (m *jobModel) Items() interface{} {
	return m.items
}

func (m *jobModel) initJobs(ni *walk.NotifyIcon) {
	m.items, _ = loadJobs()
	m.PublishRowsReset()
	m.updateJobs(ni)
}

func (m *jobModel) updateJobs(ni *walk.NotifyIcon) {
	jenkinsURLs := getJobsURLs()
	jobs := getJobsFromMultiple(jenkinsURLs)
	items := make([]*job, len(m.items))
	copy(items, m.items)
	for i := 0; i < len(items); i++ {
		found := false
		oldJob := m.items[i]
		var newJob *job
		for j := 0; j < len(jobs.Jobs); j++ {
			if items[i].Name == jobs.Jobs[j].Name &&
				items[i].Jenkins == jobs.Jobs[j].Jenkins {
				found = true
				newJob = jobs.Jobs[j]
				break
			}
		}
		if !found {
			items[i] = &job{
				Name: m.items[i].Name,
				URL:  strings.ReplaceAll(strings.ToLower(jenkinsURLs[0]), "/cc.xml", "/job/"+m.items[i].Name),
			}
		} else {
			items[i] = newJob
			oldStatus := oldJob.LastCompletedBuild.Result
			newStatus := newJob.LastCompletedBuild.Result
			if ni != nil && oldJob.LastCompletedBuild.Label < newJob.LastCompletedBuild.Label {
				appName := walk.App().ProductName()
				switch oldStatus {
				case "SUCCESS":
					switch newStatus {
					case "SUCCESS":
						if getSuccessiveSuccessful() {
							ni.ShowInfo(appName, newJob.Name+" is still successful.")
						}
					case "UNSTABLE":
						ni.ShowWarning(appName, newJob.Name+" has become unstable.")
					case "FAILURE":
						ni.ShowError(appName, newJob.Name+" failed.")
					}
				case "UNSTABLE":
					switch newStatus {
					case "SUCCESS":
						ni.ShowInfo(appName, newJob.Name+" is successful again.")
					case "UNSTABLE":
						ni.ShowWarning(appName, newJob.Name+" is still unstable.")
					case "FAILURE":
						ni.ShowError(appName, newJob.Name+" failed.")
					}
				case "FAILURE":
					switch newStatus {
					case "SUCCESS":
						ni.ShowInfo(appName, newJob.Name+" is successful again.")
					case "UNSTABLE":
						ni.ShowWarning(appName, newJob.Name+" is at least unstable now.")
					case "FAILURE":
						ni.ShowError(appName, newJob.Name+" still failing.")
					}
				}
			}
		}
	}

	var changedIdx []int
	for idx, item := range m.items {
		var foundItem *job
		for _, item2 := range items {
			if item.Name == item2.Name && item.Jenkins == item2.Jenkins {
				foundItem = item2
				break
			}
		}
		if foundItem != nil {
			changed := false
			if item.LastBuild != foundItem.LastBuild {
				item.LastBuild = foundItem.LastBuild
				changed = true
			}
			if item.LastCompletedBuild != foundItem.LastCompletedBuild {
				item.LastCompletedBuild = foundItem.LastCompletedBuild
				changed = true
			}
			if item.URL != foundItem.URL {
				item.URL = foundItem.URL
				changed = true
			}
			if changed {
				changedIdx = append(changedIdx, idx)
			}
		}
	}

	if len(changedIdx) <= 5 {
		for _, idx := range changedIdx {
			m.PublishRowChanged(idx)
		}
	} else {
		m.PublishRowsReset()
	}
}

type jenkinsMainWindow struct {
	*walk.MainWindow
	table  *walk.TableView
	ticker *time.Ticker
}

func doExit() {
	walk.App().ActiveForm().Synchronize(func() {
		walk.App().ActiveForm().AsFormBase().Close()
		walk.App().Exit(0)
	})
}

func (mw *jenkinsMainWindow) reInit() {
	settings := walk.App().Settings()
	var interval int
	intervalStr, ok := settings.Get("Interval")
	if !ok {
		interval = 10
	} else {
		interval, _ = strconv.Atoi(intervalStr)
	}
	mw.ticker = time.NewTicker(time.Duration(interval) * time.Second)
	mw.table.Model().(*jobModel).initJobs(nil)
}

func (mw *jenkinsMainWindow) WndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if msg == win.WM_SYSCOMMAND && win.SC_MINIMIZE == uint32(wParam) {
		mw.Hide()
		return 0
	}
	return mw.MainWindow.WndProc(hwnd, msg, wParam, lParam)
}
