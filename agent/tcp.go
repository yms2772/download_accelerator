package agent

import (
	"encoding/json"
	"log"
	"net"
	"sync"
	"time"
)

type tcpData struct {
	ID      string
	Conn    net.Conn
	Decoder *json.Decoder
}

func newConnection(id, ip, port string, preConn ...net.Conn) *tcpData {
	log.Println("Wait for new connection...")
	if len(preConn) != 0 {
		_ = preConn[0].Close()
	}

	conn, err := net.DialTimeout("tcp", ip+":"+port, 5*time.Second)
	for err != nil {
		conn, err = net.DialTimeout("tcp", ip+":"+port, 5*time.Second)
		if err != nil {
			if conn != nil {
				_ = conn.Close()
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	log.Printf("TCP connected: %s <> %s", conn.LocalAddr(), conn.RemoteAddr())
	return &tcpData{ID: id, Conn: conn, Decoder: json.NewDecoder(conn)}
}

func makeResponse(data networkResponse) []byte {
	jsonData, _ := json.Marshal(data)
	return jsonData
}

func (t *tcpData) sendResponse(data networkResponse) {
	data.ID = t.ID
	jsonData := makeResponse(data)

	if data.Command == upload {
		t.sendResponse(networkResponse{
			Command: progress,
			Progress: progressResponse{
				Command: splitTransfer,
				Text:    "Receiving data from client...",
			},
		})

		limit := int64(data.Settings.SplitTransferSetting.ChunkSize * 1000 * 1000)
		total := int64(len(jsonData))
		count := int64(0)
		var splitData []byte
		var splitMergedData [][]byte
		for i := int64(0); i < total; i++ {
			splitData = append(splitData, jsonData[i])
			if int64(len(splitData)) > limit || i == total-1 {
				count++
				splitMergedData = append(splitMergedData, splitData)
				splitData = []byte{}
			}
		}

		_, _ = t.Conn.Write(makeResponse(networkResponse{
			ID:      t.ID,
			Command: splitTransfer,
			SplitTransfer: splitTransferResponse{
				Index: -1,
				Total: count,
				Done:  false,
			},
		}))

		wgCount := 0
		wg := new(sync.WaitGroup)
		for i := int64(0); i < count; i++ {
			wgCount++
			wg.Add(1)
			go func(index int64) {
				defer wg.Done()
				_, _ = t.Conn.Write(makeResponse(networkResponse{
					ID:      t.ID,
					Command: splitTransfer,
					SplitTransfer: splitTransferResponse{
						Index: index,
						Total: count,
						Data:  splitMergedData[index],
						Done:  false,
					},
				}))
			}(i)
			if wgCount%data.Settings.SplitTransferSetting.ChunkParallel == 0 {
				wg.Wait()
			}
		}
		wg.Wait()
		_, _ = t.Conn.Write(makeResponse(networkResponse{
			ID:      t.ID,
			Command: splitTransfer,
			SplitTransfer: splitTransferResponse{
				Index: count,
				Total: count,
				Done:  true,
			},
		}))
	} else {
		_, _ = t.Conn.Write(jsonData)
	}
}
