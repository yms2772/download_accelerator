package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/yms2772/download_accelerator/cmd"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/dustin/go-humanize"
)

type uploadResult struct {
	ID   int
	Data []byte
}

type connectionData struct {
	Upload         []uploadResult
	Conn           net.Conn
	LastConnection time.Time
}

var (
	startTime         time.Time
	connections       = make(map[string]*connectionData)
	splitTransferData = make(map[string][][]byte)
)

func decompress(compressed [][]byte) [][]byte {
	wg := new(sync.WaitGroup)
	result := make([][]byte, len(compressed))
	for i, data := range compressed {
		wg.Add(1)
		gz, err := gzip.NewReader(bytes.NewBuffer(data))
		if err != nil {
			return nil
		}
		go func(reader *gzip.Reader, n int) {
			defer wg.Done()
			result[n], _ = io.ReadAll(reader)
		}(gz, i)
	}
	wg.Wait()
	return result
}

func (m *mainAppData) newConnection(conn net.Conn) {
	d := json.NewDecoder(conn)
MAIN:
	for {
		var resp networkResponse
		if err := d.Decode(&resp); err != nil {
			break
		}

		if len(resp.ID) == 0 {
			continue
		}

		switch resp.Command {
		case errorOccurred:
			m.Processing.Hide()
			dialog.ShowError(resp.Error, m.Window)
			continue
		case keepAlive:
			if _, ok := connections[resp.ID]; ok {
				connections[resp.ID].LastConnection = time.Now()
				continue
			}
			log.Printf("Connected: %s", resp.ID)
			connections[resp.ID] = &connectionData{
				Conn: conn,
			}
			m.Client.Content.(*fyne.Container).Add(widget.NewCheck(resp.ID, func(b bool) {}))
			m.Client.Refresh()
		case splitTransfer:
			if resp.SplitTransfer.Done {
				var merged []byte
				for _, data := range splitTransferData[resp.ID] {
					merged = append(merged, data...)
				}

				var mergedData networkResponse
				if err := json.Unmarshal(merged, &mergedData); err != nil {
					dialog.ShowError(resp.Error, m.Window)
					continue
				}

				if connections[mergedData.ID] == nil {
					connections[mergedData.ID] = &connectionData{}
				}

				if len(connections[mergedData.ID].Upload) != len(mergedData.Upload) {
					connections[mergedData.ID].Upload = make([]uploadResult, len(mergedData.Upload))
				}

				for i, upload := range mergedData.Upload {
					decompressed := decompress(upload.Data)
					if decompressed == nil {
						dialog.ShowError(errors.New("decompress failed"), m.Window)
						continue MAIN
					}
					for _, data := range decompressed {
						connections[mergedData.ID].Upload[i].Data = append(connections[mergedData.ID].Upload[i].Data, data...)
					}
					connections[mergedData.ID].Upload[i].ID = upload.ID
				}

				for _, connection := range connections {
					if len(connection.Upload) == 0 {
						continue MAIN
					}
					for _, data := range connection.Upload {
						if len(data.Data) == 0 {
							continue MAIN
						}
					}
				}

				mergedFileData := make([][][]byte, len(connections))
				for _, connection := range connections {
					for j, data := range connection.Upload {
						if len(mergedFileData[data.ID]) == 0 {
							mergedFileData[data.ID] = make([][]byte, len(mergedData.Upload))
						}
						mergedFileData[data.ID][j] = data.Data
					}
				}

				totalData := make([][]byte, len(mergedData.Upload))
				for _, data := range mergedFileData {
					for i := 0; i < len(mergedData.Upload); i++ {
						totalData[i] = append(totalData[i], data[i]...)
					}
				}

				if len(totalData) == 0 {
					continue
				}

				_ = os.Mkdir("downloaded", os.ModePerm)
				for i := 0; i < len(mergedData.Upload); i++ {
					_ = os.WriteFile("downloaded/"+mergedData.Upload[i].Filename, totalData[i], os.ModePerm)
				}

				switch mergedData.Upload[0].Type {
				case generalFile:
				case youtubeVideo:
					if len(totalData) == 2 {
						ffmpeg, ok := checkFFmpeg()
						if !ok {
							dialog.ShowError(errors.New("ffmpeg does not exist"), m.Window)
							continue
						}

						if err := cmd.PrepareBackgroundCommand(exec.Command(ffmpeg, "-y",
							"-i", "downloaded/"+mergedData.Upload[0].Filename,
							"-i", "downloaded/"+mergedData.Upload[1].Filename,
							"-c:v", "copy",
							"-c:a", "copy",
							"-shortest",
							"downloaded/youtube_with_audio.mp4",
							"-loglevel", "warning",
						)).Run(); err != nil {
							dialog.ShowError(errors.New("cannot merge audio"), m.Window)
							continue
						}

						for i := 0; i < len(mergedData.Upload); i++ {
							_ = os.Remove("downloaded/" + mergedData.Upload[i].Filename)
						}
					}
				}

				for _, item := range m.Log {
					objects := item.Content.(*fyne.Container).Objects
					if len(objects) == 0 {
						continue
					}
					objects[0].(*widget.Card).SetSubTitle("Download complete")
					objects[0].(*widget.Card).Content.(*widget.ProgressBar).SetValue(1)
				}
				m.Processing.Hide()
				dialog.ShowInformation("Done", fmt.Sprintf("Download complete\nElapsed time: %s", durationFormat(time.Now().Sub(startTime).Seconds())), m.Window)
				continue
			}

			if resp.SplitTransfer.Index == -1 {
				splitTransferData[resp.ID] = make([][]byte, resp.SplitTransfer.Total)
				connections[resp.ID].Upload = []uploadResult{}
				continue
			}

			card := m.Log[resp.ID].Content.(*fyne.Container).Objects[0].(*widget.Card)
			nowProgress := float64(resp.SplitTransfer.Index) / float64(resp.SplitTransfer.Total)
			if card.Content.(*widget.ProgressBar).Value < nowProgress {
				card.Content.(*widget.ProgressBar).SetValue(nowProgress)
			}
			splitTransferData[resp.ID][resp.SplitTransfer.Index] = resp.SplitTransfer.Data
		case progress:
			objects := m.Log[resp.ID].Content.(*fyne.Container).Objects
			switch resp.Progress.Command {
			case download:
				if len(objects) < resp.Progress.ID {
					continue
				}
				var sum int64
				for _, item := range resp.Progress.NetworkUsage {
					sum += item
				}
				m.LogWindow.SetTitle(fmt.Sprintf("LogViewer - %s/s", humanize.Bytes(uint64(sum))))
				card := objects[resp.Progress.ID].(*widget.Card)
				card.SetSubTitle(resp.Progress.Text)
				switch card.Content.(type) {
				case *widget.ProgressBarInfinite:
					card.SetContent(widget.NewProgressBar())
				}
				card.Content.(*widget.ProgressBar).SetValue(resp.Progress.Percent)
			case splitTransfer:
				m.Log[resp.ID].Content.(*fyne.Container).RemoveAll()
				m.Log[resp.ID].Content.(*fyne.Container).Add(widget.NewCard("", resp.Progress.Text, widget.NewProgressBar()))
				m.Log[resp.ID].Content.(*fyne.Container).Objects[0].(*widget.Card).SetSubTitle(resp.Progress.Text)
			case compress:
				m.LogWindow.SetTitle("LogViewer")
				objects[resp.Progress.ID].(*widget.Card).SetSubTitle(resp.Progress.Text)
				objects[resp.Progress.ID].(*widget.Card).SetContent(widget.NewProgressBarInfinite())
			}
		}
	}
}

func sendResponse(data networkResponse) {
	jsonData, _ := json.Marshal(data)
	n, err := connections[data.ID].Conn.Write(jsonData)
	if err == nil {
		log.Printf("write %d byte(s)", n)
	}
}
