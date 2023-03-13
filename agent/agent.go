package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

type RunAgentOptions struct {
	ID   string
	IP   string
	Port string
}

type Data struct {
	Ctx    context.Context
	Cancel context.CancelFunc
}

func New() *Data {
	data := new(Data)
	data.Ctx, data.Cancel = context.WithCancel(context.Background())
	return data
}

func (d *Data) RunAgent(opts ...RunAgentOptions) error {
	var id, ip, port string
	if len(opts) != 0 {
		id = opts[0].ID
		ip = opts[0].IP
		port = opts[0].Port
	} else {
		id = os.Getenv("DOWNLOAD_ACCELERATOR_ID")
		ip = os.Getenv("DOWNLOAD_ACCELERATOR_IP")
		port = os.Getenv("DOWNLOAD_ACCELERATOR_PORT")
	}
	if len(id) == 0 {
		id = fmt.Sprintf("daa_%d", time.Now().UnixNano())
	}
	if len(ip) == 0 || len(port) == 0 {
		return errors.New("check environemnts")
	}

	stop := false
	tcp := newConnection(id, ip, port)

	go func() {
		for !stop {
			tcp.sendResponse(networkResponse{
				Command:   keepAlive,
				KeepAlive: keepAliveResponse{Command: keepAlive},
			})
			time.Sleep(500 * time.Millisecond)
		}
	}()

	go func() {
		for !stop {
			var resp networkResponse
			if err := tcp.Decoder.Decode(&resp); err != nil {
				tcp = newConnection(id, ip, port, tcp.Conn)
				continue
			}

			switch resp.Command {
			case download:
				data, err := tcp.download(resp.Download)
				if err != nil {
					tcp.sendResponse(networkResponse{Command: errorOccurred, Error: err})
					continue
				}

				var uploadResp []uploadResponse
				for i, item := range data {
					uploadResp = append(uploadResp, uploadResponse{
						Type:     resp.Download[i].Type,
						ID:       resp.Download[i].ID,
						Filename: resp.Download[i].Filename,
						Data:     item,
					})
				}

				tcp.sendResponse(networkResponse{
					Command:  upload,
					Upload:   uploadResp,
					Settings: resp.Settings,
				})
			}
		}
	}()

	<-d.Ctx.Done()
	stop = true
	return nil
}
