package agent

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
)

type downloader struct {
	io.Reader

	TCP           *tcpData
	ProgressSent  time.Time
	Index         int
	ContentLength int64
	Total         int64
	PrevTotal     int64
}

var networkUsage []int64

func (d *downloader) Read(p []byte) (int, error) {
	n, err := d.Reader.Read(p)
	go func() {
		d.Total += int64(n)
		if err == nil && time.Now().Sub(d.ProgressSent).Seconds() > 1 {
			d.ProgressSent = time.Now()
			networkUsage[d.Index] = d.Total - d.PrevTotal
			d.PrevTotal = d.Total
			d.TCP.sendResponse(networkResponse{
				Command: progress,
				Progress: progressResponse{
					ID:           d.Index,
					Command:      download,
					Text:         fmt.Sprintf("Downloading... %s/s", humanize.Bytes(uint64(networkUsage[d.Index]))),
					Percent:      float64(d.Total) / float64(d.ContentLength),
					NetworkUsage: networkUsage,
				},
			})
		}
	}()
	return n, err
}

func (d *downloader) Close() error {
	return d.Close()
}

func (t *tcpData) download(responses []downloadResponse) ([][][]byte, error) {
	client := &http.Client{}
	result := make([][][]byte, len(responses))
	for i, resp := range responses {
		result[i] = make([][]byte, resp.Connection)
		wg := new(sync.WaitGroup)
		total := resp.LastIndex - resp.StartIndex
		parts := total / int64(resp.Connection)
		networkUsage = make([]int64, resp.Connection)
		for j := 0; j < resp.Connection; j++ {
			wg.Add(1)
			nextParts := resp.StartIndex + parts
			if j == resp.Connection-1 {
				nextParts = resp.LastIndex
			}
			go t.getPart(wg, client, &result[i][j], j, resp.StartIndex, nextParts, resp.URL)
			resp.StartIndex = nextParts + 1
		}
		wg.Wait()

		for _, item := range result[i] {
			if len(item) == 0 {
				return nil, errors.New("download is not completely done")
			}
		}
	}
	return result, nil
}

func (t *tcpData) getPart(wg *sync.WaitGroup, client *http.Client, data *[]byte, index int, start, last int64, uri string) {
	defer wg.Done()

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return
	}

	req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", start, last))

	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return
	}

	resp.Body = &downloader{
		TCP:           t,
		ProgressSent:  time.Now(),
		Index:         index,
		Reader:        resp.Body,
		ContentLength: last - start,
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	t.sendResponse(networkResponse{
		Command: progress,
		Progress: progressResponse{
			ID:      index,
			Command: compress,
			Text:    "Compressing...",
		},
	})

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write(body)
	_ = gz.Close()

	*data = buf.Bytes()

	t.sendResponse(networkResponse{
		Command: progress,
		Progress: progressResponse{
			ID:      index,
			Command: download,
			Text:    "Download complete",
			Percent: 1,
		},
	})
}
