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
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type connectionData struct {
	uploadResponse

	Conn net.Conn
}

var (
	startTime         time.Time
	connections       = make(map[string]*connectionData)
	splitTransferData = make(map[string][][]byte)
)

func (m *mainAppData) newConnection(conn net.Conn) {
	d := json.NewDecoder(conn)
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
			dialog.ShowError(resp.Error, m.Window)
			continue
		case keepAlive:
			if _, ok := connections[resp.ID]; ok {
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

				gz, err := gzip.NewReader(bytes.NewBuffer(mergedData.Upload.Data))
				if err != nil {
					dialog.ShowError(errors.New("decompress failed"), m.Window)
					continue
				}

				connections[mergedData.ID].Data, err = io.ReadAll(gz)
				if err != nil {
					dialog.ShowError(errors.New("decompress failed"), m.Window)
					continue
				}

				connections[mergedData.ID].ID = mergedData.Upload.ID
				done := true
				for _, data := range connections {
					if len(data.Data) == 0 {
						done = false
						break
					}
				}

				if !done {
					continue
				}

				mergedFileData := make([][]byte, len(connections))
				for _, data := range connections {
					mergedFileData[data.ID] = data.Data
				}

				var totalData []byte
				for _, data := range mergedFileData {
					totalData = append(totalData, data...)
				}

				_ = os.Mkdir("downloaded", os.ModePerm)
				_ = os.WriteFile("downloaded/"+mergedData.Upload.Filename, totalData, os.ModePerm)

				for _, item := range m.Log {
					objects := item.Content.(*fyne.Container).Objects
					if len(objects) == 0 {
						continue
					}
					objects[0].(*widget.Card).SetSubTitle("Download complete")
					objects[0].(*widget.Card).Content.(*widget.ProgressBar).SetValue(1)
				}
				dialog.ShowInformation("Done", fmt.Sprintf("Download complete\nElapsed time: %s", durationFormat(time.Now().Sub(startTime).Seconds())), m.Window)
				continue
			}

			if resp.SplitTransfer.Index == -1 {
				splitTransferData[resp.ID] = make([][]byte, resp.SplitTransfer.Total)
				continue
			}

			card := m.Log[resp.ID].Content.(*fyne.Container).Objects[0].(*widget.Card)
			card.SetSubTitle("Receiving data from client...")

			switch card.Content.(type) {
			case *widget.ProgressBarInfinite:
				card.Content = widget.NewProgressBar()
			}

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
				card := objects[resp.Progress.ID].(*widget.Card)
				card.SetSubTitle(resp.Progress.Text)
				card.Content.(*widget.ProgressBar).SetValue(resp.Progress.Percent)
			case compress:
				m.Log[resp.ID].Content.(*fyne.Container).RemoveAll()
				m.Log[resp.ID].Content.(*fyne.Container).Add(widget.NewCard("", resp.Progress.Text, widget.NewProgressBarInfinite()))
				m.Log[resp.ID].Content.(*fyne.Container).Objects[0].(*widget.Card).SetSubTitle(resp.Progress.Text)
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
