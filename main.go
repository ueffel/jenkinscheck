//go:generate goversioninfo.exe -o rsrc.syso
// JenkinsCheck project main.go
package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
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
	DEBUG := false
	var proxy func(*http.Request) (*url.URL, error)
	if DEBUG {
		proxyURL, _ := url.Parse("http://127.0.0.1:8888")
		proxy = http.ProxyURL(proxyURL)
	} else {
		proxy = http.ProxyFromEnvironment
	}
	trans := &http.Transport{Proxy: proxy}
	http.DefaultTransport = trans

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
	settings.Load()
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
				AssignTo:              &mainWindow.table,
				Name:                  "tableView",
				AlternatingRowBGColor: walk.RGB(240, 240, 240),
				ColumnsOrderable:      true,
				Columns: []TableViewColumn{
					{
						Name:  "Status",
						Width: 70,
					},
					{
						Name:  "Name",
						Width: 400,
					},
					{
						Name: "Label",
					},
					{
						Name: "Activity",
					},
					{
						Name:   "BuildTime",
						Format: "2006-01-02 15:04:05",
						Width:  150,
					},
				},
				Model:           tableModel,
				CustomRowHeight: 30,
				StyleCell: func(style *walk.CellStyle) {
					item := tableModel.items[style.Row()]

					if style.Col() < 0 {
						return
					}

					switch mainWindow.table.Columns().At(style.Col()).Name() {
					case "Status":
						style.Font = boldFont
						switch item.Status {
						case "Success":
							canvas := style.Canvas()
							if canvas != nil {
								canvas.GradientFillRectangle(walk.RGB(100, 200, 100), walk.RGB(200, 250, 200), walk.Horizontal, style.Bounds())
								canvas.DrawText("👍", boldFont, walk.RGB(0, 0, 0), style.Bounds(), 127)
							}
						case "Unstable":
							canvas := style.Canvas()
							if canvas != nil {
								canvas.GradientFillRectangle(walk.RGB(150, 150, 0), walk.RGB(250, 250, 0), walk.Horizontal, style.Bounds())
								canvas.DrawText("👀", boldFont, walk.RGB(0, 0, 0), style.Bounds(), 127)
							}
						case "Failure":
							canvas := style.Canvas()
							if canvas != nil {
								canvas.GradientFillRectangle(walk.RGB(200, 0, 0), walk.RGB(100, 0, 0), walk.Horizontal, style.Bounds())
								canvas.DrawText("👎", boldFont, walk.RGB(0, 0, 0), style.Bounds(), 127)
							}
						case "":
						default:
							canvas := style.Canvas()
							if canvas != nil {
								canvas.FillRectangle(solidWhite, style.Bounds())
								canvas.DrawText("🤷‍", boldFont, walk.RGB(0, 0, 0), style.Bounds(), 127)
							}
						}
					case "Activity":
						switch item.Activity {
						case "Sleeping":
							canvas := style.Canvas()
							if canvas != nil {
								canvas.DrawText("💤", boldFont, walk.RGB(0, 0, 0), style.Bounds(), 127)
							}
						case "Building":
							canvas := style.Canvas()
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

	tableModel.initJobs()
	go tableModel.updateJobs(ni)

	interval := getInterval()

	mainWindow.ticker = time.NewTicker(time.Duration(interval) * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-mainWindow.ticker.C:
				tableModel.updateJobs(ni)
			case <-quit:
				mainWindow.ticker.Stop()
			}
		}
	}()
	defer close(quit)

	walk.InitWrapperWindow(mainWindow)
	mainWindow.Run()
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

func (m *jobModel) initJobs() {
	m.items = loadJobs()
	m.PublishRowsReset()
}

func (m *jobModel) updateJobs(ni *walk.NotifyIcon) {
	log.Println("Updating")
	ccURL := getCCXmlURL()
	jobs := getCCXmlJobs(ccURL)
	items := make([]*job, len(m.items))
	copy(items, m.items)
	for i := 0; i < len(items); i++ {
		found := false
		oldJob := m.items[i]
		var newJob *job
		for j := 0; j < len(jobs.Jobs); j++ {
			if items[i].Name == jobs.Jobs[j].Name {
				found = true
				newJob = &jobs.Jobs[j]
				break
			}
		}
		if !found {
			items[i] = &job{
				Name: m.items[i].Name,
				URL:  strings.ReplaceAll(strings.ToLower(ccURL), "/cc.xml", "/job/"+m.items[i].Name),
			}
		} else {
			newJob.BuildTime = newJob.BuildTime.Local()
			items[i] = newJob
			go func() {
				if newJob.Status == "Failure" {
					number, err := getLastUnstable(newJob)
					if err != nil {
						log.Println(err)
						number = -1
					}
					if newJob.Label == number {
						newJob.Status = "Unstable"
					}
				}
				oldStatus := oldJob.Status
				newStatus := newJob.Status
				if ni != nil && oldJob.Label < newJob.Label {
					appName := walk.App().ProductName()
					switch oldStatus {
					case "Success":
						switch newStatus {
						case "Success":
							if getSuccessiveSuccessful() {
								ni.ShowInfo(appName, newJob.Name+" is still successful.")
							}
						case "Unstable":
							ni.ShowWarning(appName, newJob.Name+" has become unstable.")
						case "Failure":
							ni.ShowError(appName, newJob.Name+" failed.")
						}
					case "Unstable":
						switch newStatus {
						case "Success":
							ni.ShowInfo(appName, newJob.Name+" is successful again.")
						case "Unstable":
							ni.ShowWarning(appName, newJob.Name+" is still unstable.")
						case "Failure":
							ni.ShowError(appName, newJob.Name+" failed.")
						}
					case "Failure":
						switch newStatus {
						case "Success":
							ni.ShowInfo(appName, newJob.Name+" is successful again.")
						case "Unstable":
							ni.ShowWarning(appName, newJob.Name+" is at least unstable now.")
						case "Failure":
							ni.ShowError(appName, newJob.Name+" still failing.")
						}
					}
				}
			}()
		}
	}
	m.items = items
	m.PublishRowsReset()
}

type jenkinsMainWindow struct {
	*walk.MainWindow
	table  *walk.TableView
	ticker *time.Ticker
}

func doExit() {
	walk.App().ActiveForm().AsFormBase().Close()
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
	mw.table.Model().(*jobModel).initJobs()
	mw.table.Model().(*jobModel).updateJobs(nil)
}

func (mw *jenkinsMainWindow) WndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if msg == win.WM_SYSCOMMAND && win.SC_MINIMIZE == uint32(wParam) {
		mw.Hide()
		return 0
	}
	return mw.MainWindow.WndProc(hwnd, msg, wParam, lParam)
}
