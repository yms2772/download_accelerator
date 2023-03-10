package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/dustin/go-humanize"
	"github.com/kkdai/youtube/v2"
	"github.com/yms2772/download_accelerator/agent"
)

type mainAppData struct {
	W, H       float32
	App        fyne.App
	Window     fyne.Window
	LogWindow  fyne.Window
	Client     *container.Scroll
	Log        map[string]*container.Scroll
	Processing *dialog.ProgressInfiniteDialog
	SelfClient *agent.Data
	Connected  bool
}

func (m *mainAppData) refreshClient() {
	for id, conn := range connections {
		if time.Now().Sub(conn.LastConnection).Seconds() >= 1 {
			delete(connections, id)
			for _, object := range m.Client.Content.(*fyne.Container).Objects {
				switch object.(type) {
				case *widget.Check:
					if object.(*widget.Check).Text == id {
						object.Hide()
					}
				}
			}
		}
	}
}

func main() {
	runMode := flag.String("mode", "downloader", "run mode: 'downloader', 'client' (default: 'downloader')")
	flag.Parse()

	switch *runMode {
	case "downloader":
	case "client":
		agentData := agent.New()
		if err := agentData.RunAgent(); err != nil {
			log.Fatal(err)
		}
	default:
		flag.PrintDefaults()
		return
	}

	mainApp := new(mainAppData)
	mainApp.App = app.NewWithID("download_accelerator")
	mainApp.App.Settings().SetTheme(&myTheme{})

	mainApp.W, mainApp.H = 750, 400
	mainApp.Window = mainApp.App.NewWindow("Download Accelerator")
	mainApp.Window.Resize(fyne.NewSize(mainApp.W, mainApp.H))
	mainApp.Window.SetFixedSize(true)
	mainApp.Window.SetMaster()

	mainApp.Client = container.NewVScroll(container.NewVBox())

	allCheck := widget.NewCheck("All", func(b bool) {
		go func() {
			if !mainApp.Connected {
				dialog.ShowError(errors.New("tcp server is closed"), mainApp.Window)
				return
			}

			for _, object := range mainApp.Client.Content.(*fyne.Container).Objects {
				switch object.(type) {
				case *widget.Check:
					object.(*widget.Check).SetChecked(b)
				}
			}
		}()
	})

	selfCheck := widget.NewCheck("Self", func(b bool) {
		go func() {
			if !mainApp.Connected {
				dialog.ShowError(errors.New("tcp server is closed"), mainApp.Window)
				return
			}

			if b {
				mainApp.SelfClient = agent.New()
				mainApp.Processing.Show()

				go func() {
					for {
						time.Sleep(time.Second)
						if _, ok := connections["self_client"]; ok {
							for _, object := range mainApp.Client.Content.(*fyne.Container).Objects {
								switch object.(type) {
								case *widget.Check:
									if object.(*widget.Check).Text == "self_client" {
										object.(*widget.Check).SetChecked(b)
									}
								}
							}
							mainApp.Processing.Hide()
							return
						}
					}
				}()

				if err := mainApp.SelfClient.RunAgent(agent.RunAgentOptions{
					ID:   "self_client",
					IP:   "127.0.0.1",
					Port: mainApp.App.Preferences().StringWithFallback("data_transform_port", "8001"),
				}); err != nil {
					dialog.ShowError(errors.New("an error occurred:\n"+err.Error()), mainApp.Window)
				}
			} else {
				if mainApp.SelfClient == nil {
					dialog.ShowError(errors.New("cannot stop the self client"), mainApp.Window)
					return
				}
				mainApp.SelfClient.Cancel()
			}
		}()
	})

	allCheck.Disable()
	selfCheck.Disable()

	mainApp.Client.Content.(*fyne.Container).Add(container.NewHBox(allCheck, selfCheck))

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for range ticker.C {
			mainApp.refreshClient()
		}
	}()

	mainApp.Log = make(map[string]*container.Scroll)
	mainApp.Processing = dialog.NewProgressInfinite("Process", "Processing...", mainApp.Window)

	logCard := widget.NewCard("", "", container.NewVScroll(container.NewVBox()))

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

	clientConnectBtn := widget.NewButtonWithIcon("", theme.SearchIcon(), nil)
	clientConnectBtn.OnTapped = func() {
		go func() {
			mainApp.Processing.Show()
			defer mainApp.Processing.Hide()

			go func() {
				clientConnectBtn.Disable()
				allCheck.Enable()
				selfCheck.Enable()
				defer func() {
					clientConnectBtn.Enable()
					allCheck.Disable()
					selfCheck.Disable()
					mainApp.Connected = false
				}()

				l, err := net.Listen("tcp", "0.0.0.0:"+mainApp.App.Preferences().StringWithFallback("data_transform_port", "8001"))
				if nil != err {
					dialog.ShowError(errors.New(fmt.Sprintf("cannot open tcp server on %s port", mainApp.App.Preferences().StringWithFallback("data_transform_port", "8001"))), mainApp.Window)
					return
				}
				defer l.Close()

				mainApp.Connected = true

				for {
					conn, err := l.Accept()
					if nil != err {
						dialog.ShowError(errors.New("connection refused"), mainApp.Window)
						continue
					}

					go mainApp.newConnection(conn)
				}
			}()
		}()
	}

	clientConnectBox := container.NewBorder(nil, nil, nil, clientConnectBtn, widget.NewForm(widget.NewFormItem("Port", clientPortInput)), clientConnectBtn)

	filenameInput := widget.NewEntry()
	filenameInput.Validator = func(s string) error {
		if len(s) == 0 {
			return errors.New("required")
		}
		return nil
	}

	sizeLabel := widget.NewLabel("")

	var downResp []downloadResponse
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
			progressDlg := dialog.NewProgressInfinite("Connect", "Connecting...", mainApp.Window)
			progressDlg.Resize(fyne.NewSize(300, 0))
			progressDlg.Show()
			defer progressDlg.Hide()

			u, err := url.Parse(urlInput.Text)
			if err != nil {
				return
			}
			switch u.Host {
			case "www.youtube.com", "youtube.com", "youtu.be":
				yt := youtube.Client{}
				video, err := yt.GetVideo(urlInput.Text)
				if err != nil {
					dialog.ShowError(errors.New("invalid youtube url"), mainApp.Window)
					return
				}

				qualitySelect := widget.NewSelect(youtubeQuality(video.Formats), nil)
				qualitySelect.SetSelectedIndex(0)

				if len(video.Thumbnails) == 0 {
					dialog.ShowError(errors.New("thumbnail not found"), mainApp.Window)
					return
				}

				thumbnailData := video.Thumbnails[len(video.Thumbnails)-1]
				resp, err := http.Get(thumbnailData.URL)
				if resp != nil {
					defer resp.Body.Close()
				}
				if err != nil {
					dialog.ShowError(errors.New("cannot get a thumbnail"), mainApp.Window)
					return
				}

				body, _ := io.ReadAll(resp.Body)

				thumbnailCanvas := canvas.NewImageFromResource(fyne.NewStaticResource("thumbnail.jpg", body))
				thumbnailCanvas.FillMode = canvas.ImageFillStretch
				thumbnailCanvas.SetMinSize(fyne.NewSize(500, float32(thumbnailData.Height*500/thumbnailData.Width)))

				ytWindow := mainApp.App.NewWindow("YouTube")
				ytWindow.Resize(fyne.NewSize(520, 700))
				ytWindow.SetFixedSize(true)
				ytWindow.SetMainMenu(fyne.NewMainMenu(
					fyne.NewMenu("Tools",
						fyne.NewMenuItem("Download Thumbnail", func() {
							go func() {
								filename := fmt.Sprintf("thumbnail_%dx%d.jpg", thumbnailData.Width, thumbnailData.Height)
								_ = os.Mkdir("thumbnail", os.ModePerm)
								_ = os.WriteFile("thumbnail/"+filename, body, os.ModePerm)
								dialog.ShowInformation("Thumbnail Download", "Saved at: thumbnail/"+filename, ytWindow)
							}()
						}),
						fyne.NewMenuItem("Export Heat Seeker (.pbf file)", func() {
							go func() {
								dlg := dialog.NewProgressInfinite("Heat Seeker", "Exporting heat seeker from YouTube...", ytWindow)
								dlg.Show()
								defer dlg.Hide()

								hsData, err := youtubeHeatSeeker(video.ID)
								if err != nil {
									log.Print(err)
									dialog.ShowError(errors.New("cannot export the heat seeker"), ytWindow)
									return
								}

								f, err := os.OpenFile("downloaded/youtube_with_audio.pbf", os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModeAppend)
								if err != nil {
									log.Print(err)
									dialog.ShowError(errors.New("cannot create the pbf file"), ytWindow)
									return
								}
								defer f.Close()

								_, _ = f.WriteString("[PlayRepeat]\n")
								count := 0

								for i := 0; i < len(hsData.Chapter); i++ {
									var duration int64
									if i == len(hsData.Chapter)-1 {
										duration = video.Duration.Milliseconds() - hsData.Chapter[i].StartMillis
									} else {
										duration = hsData.Chapter[i+1].StartMillis - hsData.Chapter[i].StartMillis
									}
									_, _ = f.WriteString(fmt.Sprintf("%d=%d*%d*0*%s\n", count, hsData.Chapter[i].StartMillis, duration, hsData.Chapter[i].Title))
									_, _ = f.WriteString(fmt.Sprintf("%d_d=0\n", count))
									_, _ = f.WriteString(fmt.Sprintf("%d_e=0\n", count))
									count++
								}

								for i, item := range hsData.HeatSeeker {
									_, _ = f.WriteString(fmt.Sprintf("%d=%d*%d*0*No.%d\n", count, item.StartMillis, video.Duration.Milliseconds()-item.StartMillis, i+1))
									_, _ = f.WriteString(fmt.Sprintf("%d_d=0\n", count))
									_, _ = f.WriteString(fmt.Sprintf("%d_e=0\n", count))
									count++
								}
								dialog.ShowInformation("Heat Seeker", "Saved at: downloaded/youtube_with_audio.pbf", ytWindow)
							}()
						}),
					),
				))

				ytTitle := widget.NewHyperlink(video.Title, u)
				ytTitle.Wrapping = fyne.TextTruncate

				ytResolution := widget.NewLabel(fmt.Sprintf("%d x %d", video.Formats[qualitySelect.SelectedIndex()].Width, video.Formats[qualitySelect.SelectedIndex()].Height))
				ytAvgBitrate := widget.NewLabel(fmt.Sprintf("%s Kbps", humanize.Comma(int64(video.Formats[qualitySelect.SelectedIndex()].AverageBitrate/1000))))
				ytAudioIncluded := widget.NewCheck("", nil)
				ytExpectedSize := widget.NewLabel(fmt.Sprintf("~ %s", humanize.Bytes(uint64(video.Formats[qualitySelect.SelectedIndex()].ContentLength))))
				if video.Formats[qualitySelect.SelectedIndex()].AudioChannels == 0 {
					ytAudioIncluded.SetChecked(false)
					ytAudioIncluded.Enable()
				} else {
					ytAudioIncluded.SetChecked(true)
					ytAudioIncluded.Disable()
				}
				ytAudioIncluded.OnChanged = func(b bool) {
					if video.Formats[qualitySelect.SelectedIndex()].AudioChannels == 0 && b {
						dlg := dialog.NewProgressInfinite("Check", "Cheking ffmpeg...", ytWindow)
						dlg.Show()
						defer dlg.Hide()
						if _, ok := checkFFmpeg(); !ok {
							dialog.ShowError(errors.New("ffmpeg does not exist in './bin' or PATH"), ytWindow)
							ytAudioIncluded.SetChecked(false)
							return
						}
					}
				}

				qualitySelect.OnChanged = func(s string) {
					ytResolution.SetText(fmt.Sprintf("%d x %d", video.Formats[qualitySelect.SelectedIndex()].Width, video.Formats[qualitySelect.SelectedIndex()].Height))
					ytAvgBitrate.SetText(fmt.Sprintf("%s Kbps", humanize.Comma(int64(video.Formats[qualitySelect.SelectedIndex()].AverageBitrate/1000))))
					ytExpectedSize.SetText(fmt.Sprintf("~ %s", humanize.Bytes(uint64(video.Formats[qualitySelect.SelectedIndex()].ContentLength))))
					if video.Formats[qualitySelect.SelectedIndex()].AudioChannels == 0 {
						ytAudioIncluded.SetChecked(false)
						ytAudioIncluded.Enable()
					} else {
						ytAudioIncluded.SetChecked(true)
						ytAudioIncluded.Disable()
					}
				}

				ytFormSubmit := make(chan bool)
				ytDetailForm := widget.NewForm(
					widget.NewFormItem("Title", ytTitle),
					widget.NewFormItem("Author", widget.NewLabel(video.Author)),
					widget.NewFormItem("Views", widget.NewLabel(humanize.Comma(int64(video.Views)))),
					widget.NewFormItem("Duration", widget.NewLabel(durationFormat(video.Duration.Seconds()))),
					widget.NewFormItem("Resolution", ytResolution),
					widget.NewFormItem("Avg Bitrate", ytAvgBitrate),
					widget.NewFormItem("Audio Included", ytAudioIncluded),
					widget.NewFormItem("Expected Size", ytExpectedSize),
					widget.NewFormItem("Quality", qualitySelect),
				)
				ytDetailForm.SubmitText = "Apply"
				ytDetailForm.OnSubmit = func() {
					ytFormSubmit <- true
					ytWindow.Close()
				}

				ytWindow.SetContent(container.NewVScroll(container.NewBorder(thumbnailCanvas, nil, nil, nil, thumbnailCanvas, ytDetailForm)))
				ytWindow.SetOnClosed(func() {
					ytFormSubmit <- false
				})
				ytWindow.Show()

				if !<-ytFormSubmit {
					return
				}
				if ytAudioIncluded.Checked {
					audio, err := youtubeAudio(video.Formats)
					if err != nil {
						dialog.ShowError(errors.New("cannot find audio stream data"), mainApp.Window)
						return
					}
					downResp = make([]downloadResponse, 2)
					downResp[1].URL, err = yt.GetStreamURL(video, &audio)
					if err != nil {
						dialog.ShowError(errors.New("cannot get a audio stream url"), mainApp.Window)
						return
					}
					downResp[1].Filename = "audio.mp4"
					downResp[1].ContentLength = audio.ContentLength
				} else {
					downResp = make([]downloadResponse, 1)
				}

				downResp[0] = downloadResponse{Type: youtubeVideo}
				downResp[0].URL, err = yt.GetStreamURL(video, &video.Formats[qualitySelect.SelectedIndex()])
				if err != nil {
					dialog.ShowError(errors.New("cannot get a stream url"), mainApp.Window)
					return
				}
				extension, _, _ := mime.ParseMediaType(video.Formats[qualitySelect.SelectedIndex()].MimeType)
				downResp[0].Filename = "video." + strings.Split(extension, "/")[1]
				downResp[0].ContentLength = video.Formats[qualitySelect.SelectedIndex()].ContentLength
			default:
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

				downResp = make([]downloadResponse, 1)
				if _, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition")); err != nil || len(params["filename"]) == 0 {
					u, _ := url.Parse(urlInput.Text)
					paths := strings.Split(u.Path, "/")
					if len(paths) != 0 {
						downResp[0].Filename = paths[len(paths)-1]
					} else {
						downResp[0].Filename = "unknown"
					}
				} else {
					downResp[0].Filename = params["filename"]
				}
				downResp[0].URL = s
				downResp[0].ContentLength = resp.ContentLength
			}
			var totalLength int64
			for _, resp := range downResp {
				totalLength += resp.ContentLength
			}
			filenameInput.SetText(downResp[0].Filename)
			sizeLabel.SetText("~ " + humanize.Bytes(uint64(totalLength)))
		}()
	}

	parallelInput := widget.NewEntry()
	parallelInput.SetText("50")
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

	pasteURL := widget.NewButtonWithIcon("", theme.ContentPasteIcon(), func() {
		if mainApp.Window.Clipboard() == nil {
			return
		}
		urlInput.SetText(mainApp.Window.Clipboard().Content())
	})

	settingForm := widget.NewForm(
		widget.NewFormItem("URL", container.NewBorder(nil, nil, nil, pasteURL, urlInput, pasteURL)),
		widget.NewFormItem("Filename", container.NewVBox(filenameInput, sizeLabel)),
		widget.NewFormItem("Parallel", parallelInput),
		widget.NewFormItem("Chunk Size", container.NewGridWithColumns(2, chunkSizeInput, widget.NewLabelWithStyle("MB", fyne.TextAlignLeading, fyne.TextStyle{}))),
		widget.NewFormItem("Chunk Parallel", chunkParallelInput),
	)
	settingForm.SubmitText = "Download"
	settingForm.OnSubmit = func() {
		go func() {
			if !mainApp.Connected {
				dialog.ShowError(errors.New("tcp server is closed"), mainApp.Window)
				return
			}

			parallel, err := strconv.Atoi(parallelInput.Text)
			if err != nil {
				parallel = 50
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
				switch object.(type) {
				case *widget.Check:
					check := object.(*widget.Check)
					if check.Text != "All" && check.Visible() && check.Checked {
						checked = append(checked, check.Text)
						mainApp.Log[check.Text] = container.NewVScroll(container.NewVBox())
						logSelect.Options = append(logSelect.Options, check.Text)
						for i := 0; i < parallel; i++ {
							mainApp.Log[check.Text].Content.(*fyne.Container).Add(widget.NewCard("", "Preparing to download...", widget.NewProgressBar()))
						}
					}
				}
			}

			if len(checked) == 0 {
				dialog.ShowError(errors.New("no client selected"), mainApp.Window)
				return
			}

			mainApp.Processing.Show()
			logCard.SetContent(mainApp.Log[checked[0]])
			logSelect.SetSelectedIndex(0)

			logWindow := mainApp.App.NewWindow("LogViewer")
			logWindow.Resize(fyne.NewSize(400, 600))
			logWindow.SetContent(container.NewBorder(logSelectBoxBorder, nil, nil, nil,
				logSelectBoxBorder,
				logCard,
			))
			logWindow.Show()

			mainApp.LogWindow = logWindow
			startTime = time.Now()
			for i := 0; i < len(checked); i++ {
				resp := networkResponse{
					ID:      checked[i],
					Command: download,
					Settings: settingsResponse{
						SplitTransferSetting: splitTransferSettingResponse{
							ChunkSize:     chunkSize,
							ChunkParallel: chunkParallel,
						},
					},
				}

				for j := 0; j < len(downResp); j++ {
					downResp[j].ID = i
					downResp[j].Connection = parallel
					downResp[j].StartIndex = int64(i) * (downResp[j].ContentLength / int64(len(checked)))
					if i != 0 {
						downResp[j].StartIndex++
					}

					downResp[j].LastIndex = int64(i+1) * (downResp[j].ContentLength / int64(len(checked)))
					if i == len(checked)-1 {
						downResp[j].LastIndex = downResp[j].ContentLength
					}
				}

				resp.Download = downResp
				sendResponse(resp)
			}
		}()
	}

	mainApp.Window.SetContent(container.NewHBox(
		container.NewGridWrap(fyne.NewSize(mainApp.W*33.3/100, mainApp.H),
			container.NewBorder(clientConnectBox, nil, nil, nil, clientConnectBox, mainApp.Client),
		),
		container.NewGridWrap(fyne.NewSize(mainApp.W*66.6/100, mainApp.H),
			widget.NewCard("Add", "", settingForm),
		),
	))
	mainApp.Window.ShowAndRun()
}
