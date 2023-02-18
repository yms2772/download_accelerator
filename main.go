package main

import (
	"errors"
	"github.com/dustin/go-humanize"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type mainAppData struct {
	App    fyne.App
	Window fyne.Window
	Client *container.Scroll
	Log    map[string]*container.Scroll
}

func main() {
	mainApp := new(mainAppData)
	mainApp.App = app.NewWithID("download_accelerator")
	mainApp.App.Settings().SetTheme(&myTheme{})

	mainApp.Window = mainApp.App.NewWindow("Download Accelerator")
	mainApp.Window.Resize(fyne.NewSize(1280, 720))
	mainApp.Window.SetMaster()

	mainApp.Client = container.NewVScroll(container.NewVBox())
	mainApp.Client.Content.(*fyne.Container).Add(widget.NewCheck("All", func(b bool) {
		for _, object := range mainApp.Client.Content.(*fyne.Container).Objects {
			object.(*widget.Check).SetChecked(b)
		}
	}))

	mainApp.Log = make(map[string]*container.Scroll)

	go func() {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			mainApp.Client.Refresh()
		}
	}()

	logCard := widget.NewCard("Log", "", container.NewVScroll(container.NewVBox()))

	logSelect := widget.NewSelect([]string{}, func(s string) {
		logCard.SetContent(mainApp.Log[s])
	})
	logSelect.PlaceHolder = "Client ID"

	logSelectPrev := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		if logSelect.SelectedIndex() <= 0 {
			return
		}
		logSelect.SetSelectedIndex(logSelect.SelectedIndex() - 1)
		logCard.SetContent(mainApp.Log[logSelect.Selected])
	})
	logSelectNext := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
		if logSelect.SelectedIndex() == -1 || logSelect.SelectedIndex() == len(logSelect.Options)-1 {
			return
		}
		logSelect.SetSelectedIndex(logSelect.SelectedIndex() + 1)
		logCard.SetContent(mainApp.Log[logSelect.Selected])
	})

	logSelectBox := container.NewHBox(logSelectPrev, logSelectNext)
	logSelectBoxBorder := container.NewBorder(nil, nil, nil, logSelectBox,
		logSelect, logSelectBox,
	)

	clientPortInput := widget.NewEntry()
	clientPortInput.SetPlaceHolder("default: 8001")
	clientPortInput.SetText(mainApp.App.Preferences().StringWithFallback("data_transform_port", "8001"))
	clientPortInput.OnChanged = func(s string) {
		mainApp.App.Preferences().SetString("data_transform_port", s)
	}

	clientConnectBtn := widget.NewButtonWithIcon("Connect", theme.SearchIcon(), func() {
		connectContent := container.NewVBox(
			widget.NewLabel("Connect to client..."),
			widget.NewProgressBarInfinite(),
		)
		connectDialog := dialog.NewCustom("Connect", "Cancel", connectContent, mainApp.Window)
		connectDialog.SetOnClosed(func() {

		})
		connectDialog.Show()
		defer connectDialog.Hide()

		go func() {
			l, err := net.Listen("tcp", "0.0.0.0:"+mainApp.App.Preferences().StringWithFallback("data_transform_port", "8001"))
			if nil != err {
				dialog.ShowError(errors.New("cannot connect to client"), mainApp.Window)
				return
			}
			defer l.Close()

			for {
				conn, err := l.Accept()
				if nil != err {
					dialog.ShowError(errors.New("connection refused"), mainApp.Window)
					continue
				}

				go mainApp.newConnection(conn)
			}
		}()
	})

	clientConnectBox := container.NewGridWithColumns(2, widget.NewForm(widget.NewFormItem("Port", clientPortInput)), clientConnectBtn)

	filenameInput := widget.NewEntry()
	filenameInput.Validator = func(s string) error {
		if len(s) == 0 {
			return errors.New("required")
		}
		return nil
	}

	sizeLabel := widget.NewLabel("")

	var contentLength int64
	urlInput := widget.NewEntry()
	urlInput.SetPlaceHolder("https://example.com")
	urlInput.Validator = func(s string) error {
		if len(s) == 0 {
			return errors.New("required")
		}
		return nil
	}
	urlInput.OnChanged = func(s string) {
		go func() {
			progressDlg := dialog.NewCustom("Connecting...", "Cancel", widget.NewProgressBarInfinite(), mainApp.Window)
			progressDlg.Show()
			defer progressDlg.Hide()

			resp, err := http.Head(s)
			if resp != nil {
				defer resp.Body.Close()
			}
			if err != nil {
				return
			}

			if resp.StatusCode != http.StatusOK {
				return
			}
			if resp.ContentLength < 0 {
				return
			}
			if resp.ContentLength > 0 && resp.ContentLength < 1000000 {
				dialog.ShowError(errors.New("content size must be over 1mb"), mainApp.Window)
				return
			}

			var filename string
			_, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
			if err != nil || len(params["filename"]) == 0 {
				u, _ := url.Parse(urlInput.Text)
				paths := strings.Split(u.Path, "/")
				if len(paths) != 0 {
					filename = paths[len(paths)-1]
				} else {
					filename = "unknown"
				}
			} else {
				filename = params["filename"]
			}
			filenameInput.SetText(filename)
			contentLength = resp.ContentLength
			sizeLabel.SetText("~ " + humanize.Bytes(uint64(resp.ContentLength)))
		}()
	}

	parallelInput := widget.NewEntry()
	parallelInput.SetText("5")
	parallelInput.Validator = func(s string) error {
		if _, err := strconv.Atoi(s); err != nil {
			return errors.New("must enter only numbers")
		}
		return nil
	}

	chunkSizeInput := widget.NewEntry()
	chunkSizeInput.SetText("5")
	chunkSizeInput.Validator = func(s string) error {
		if _, err := strconv.Atoi(s); err != nil {
			return errors.New("must enter only numbers")
		}
		return nil
	}

	chunkParallelInput := widget.NewEntry()
	chunkParallelInput.SetText("5")
	chunkParallelInput.Validator = func(s string) error {
		if _, err := strconv.Atoi(s); err != nil {
			return errors.New("must enter only numbers")
		}
		return nil
	}

	settingForm := widget.NewForm(
		widget.NewFormItem("URL", urlInput),
		widget.NewFormItem("Filename", container.NewVBox(filenameInput, sizeLabel)),
		widget.NewFormItem("Parallel", parallelInput),
		widget.NewFormItem("Chunk Size", container.NewGridWithColumns(2, chunkSizeInput, widget.NewLabelWithStyle("MB", fyne.TextAlignLeading, fyne.TextStyle{}))),
		widget.NewFormItem("Chunk Parallel", chunkParallelInput),
	)
	settingForm.SubmitText = "Download"
	settingForm.OnSubmit = func() {
		go func() {
			parallel, err := strconv.Atoi(parallelInput.Text)
			if err != nil {
				parallel = 5
			}

			chunkSize, err := strconv.Atoi(chunkSizeInput.Text)
			if err != nil {
				chunkSize = 5
			}

			chunkParallel, err := strconv.Atoi(chunkParallelInput.Text)
			if err != nil {
				chunkParallel = 5
			}

			var checked []string
			logSelect.Options = []string{}
			for _, object := range mainApp.Client.Content.(*fyne.Container).Objects {
				check := object.(*widget.Check)
				if check.Text != "All" && check.Checked {
					checked = append(checked, check.Text)
					mainApp.Log[check.Text] = container.NewVScroll(container.NewVBox())
					logSelect.Options = append(logSelect.Options, check.Text)
					for i := 0; i < parallel; i++ {
						mainApp.Log[check.Text].Content.(*fyne.Container).Add(widget.NewCard("", "Preparing to download...", widget.NewProgressBar()))
					}
				}
			}

			if len(checked) == 0 {
				dialog.ShowError(errors.New("no clients checked"), mainApp.Window)
				return
			}

			startTime = time.Now()
			logCard.SetContent(mainApp.Log[checked[0]])
			logSelect.SetSelectedIndex(0)
			startParts := int64(0)
			parts := contentLength / int64(len(checked))

			for i := 0; i < len(checked); i++ {
				resp := networkResponse{
					ID:      checked[i],
					Command: download,
					Download: downloadResponse{
						ID:         i,
						URL:        urlInput.Text,
						Filename:   filenameInput.Text,
						Connection: parallel,
						StartIndex: startParts,
						LastIndex:  startParts + parts,
					},
					Settings: settingsResponse{
						SplitTransferSetting: splitTransferSettingResponse{
							ChunkSize:     chunkSize,
							ChunkParallel: chunkParallel,
						},
					},
				}

				if i == len(checked)-1 {
					resp.Download.LastIndex = contentLength
				}

				sendResponse(resp)
				startParts += parts + 1
			}
		}()
	}

	mainApp.Window.SetContent(container.NewGridWithColumns(3,
		container.NewBorder(clientConnectBox, nil, nil, nil,
			clientConnectBox,
			mainApp.Client,
		),
		widget.NewCard("Add", "", settingForm),
		container.NewBorder(logSelectBoxBorder, nil, nil, nil,
			logSelectBoxBorder,
			logCard,
		),
	))
	mainApp.Window.ShowAndRun()
}
