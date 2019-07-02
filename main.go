//go:generate rsrc -manifest main.manifest -arch amd64 -ico ico/favicon.ico
// JenkinsCheck project main.go
package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"time"

	"github.com/gobuffalo/packr/v2"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
	"github.com/mat/besticon/ico"
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
		proxyUrl, _ := url.Parse("http://127.0.0.1:8888")
		proxy = http.ProxyURL(proxyUrl)
	} else {
		proxy = http.ProxyFromEnvironment
	}
	trans := &http.Transport{Proxy: proxy}
	http.DefaultTransport = trans

	mainWindow := new(jenkinsMainWindow)
	boldFont, _ := walk.NewFont("Calibri", 18, walk.FontBold)
	solidWhite, _ := walk.NewSolidColorBrush(walk.RGB(255, 255, 255))

	box := packr.New("iconBox", "./ico")
	imgbytes, err := box.Find("favicon.ico")
	if err != nil {
		log.Fatal(err)
	}
	img, err := ico.Decode(bytes.NewReader(imgbytes))
	if err != nil {
		log.Fatal(err)
	}
	icon, err := walk.NewIconFromImage(img)
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
		Icon:     icon,
		Title:    appName,
		Size:     Size{900, 800},
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
				AssignTo:       &mainWindow.table,
				MultiSelection: true,
				Columns: []TableViewColumn{
					{
						DataMember: "Status",
						Width:      70,
					},
					{
						DataMember: "Name",
						Width:      400,
					},
					{
						DataMember: "Label",
					},
					{
						DataMember: "Activity",
					},
					{
						DataMember: "BuildTime",
						Format:     "2006-01-02 15:04:05",
						Width:      150,
					},
				},
				Model:           tableModel,
				CustomRowHeight: 30,
				StyleCell: func(style *walk.CellStyle) {
					item := tableModel.items[style.Row()]

					if style.Row()%2 == 1 {
						style.BackgroundColor = walk.RGB(240, 240, 240)
					}

					if style.Col() < 0 {
						return
					}

					switch mainWindow.table.Columns().At(style.Col()).DataMember() {
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
					cmd := exec.Command(exe, tableModel.items[mainWindow.table.CurrentIndex()].Url)
					cmd.Start()
				},
			},
		},
	}.Create()

	sortCol := 0
	for i := 0; i < mainWindow.table.Columns().Len(); i++ {
		if mainWindow.table.Columns().At(i).DataMember() == "Name" {
			sortCol = i
			break
		}
	}
	tableModel.Sort(sortCol, walk.SortAscending)
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
	log.Println("closing")
}

type jobModel struct {
	walk.SortedReflectTableModelBase
	items []*job
}

func (m *jobModel) Items() interface{} {
	return m.items
}

func (tableModel *jobModel) initJobs() {
	tableModel.items = loadJobs()
	tableModel.PublishRowsReset()
}

func (tableModel *jobModel) updateJobs(ni *walk.NotifyIcon) {
	log.Println("Updating")
	ccUrl := getCCXmlUrl()
	jobs := getCCXmlJobs(ccUrl)
	items := make([]*job, len(tableModel.items))
	copy(items, tableModel.items)
	for i := 0; i < len(items); i++ {
		found := false
		oldJob := tableModel.items[i]
		var newJob *job
		for j := 0; j < len(jobs.Jobs); j++ {
			if items[i].Name == jobs.Jobs[j].Name {
				found = true
				newJob = &jobs.Jobs[j]
				break
			}
		}
		if !found {
			items[i] = &job{Name: tableModel.items[i].Name, Label: "nicht gefunden!"}
		} else {
			items[i] = newJob
			go func() {
				if newJob.Status == "Failure" {
					number, err := getLastUnstable(newJob)
					if err != nil {
						log.Println(err)
						number = "-1"
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
	tableModel.items = items
	tableModel.PublishRowsReset()
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
