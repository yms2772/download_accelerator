package main

type commandType string

const (
	download      commandType = "download"
	compress      commandType = "compress"
	progress      commandType = "progress"
	errorOccurred commandType = "error"
	keepAlive     commandType = "keep_alive"
	splitTransfer commandType = "split_transfer"
)

type keepAliveResponse struct {
	Command commandType `json:"command"`
}

type downloadResponse struct {
	URL        string `json:"url"`
	ID         int    `json:"id"`
	Filename   string `json:"filename"`
	Connection int    `json:"connection"`
	StartIndex int64  `json:"start_index"`
	LastIndex  int64  `json:"last_index"`
}

type uploadResponse struct {
	ID       int    `json:"id"`
	Filename string `json:"filename"`
	Data     []byte `json:"data"`
}

type progressResponse struct {
	ID      int         `json:"id"`
	Command commandType `json:"command"`
	Text    string      `json:"text"`
	Percent float64     `json:"percent"`
}

type splitTransferResponse struct {
	Index int64  `json:"index"`
	Total int64  `json:"total"`
	Done  bool   `json:"done"`
	Data  []byte `json:"data"`
}

type splitTransferSettingResponse struct {
	ChunkSize     int `json:"chunk_size"`
	ChunkParallel int `json:"chunk_parallel"`
}

type settingsResponse struct {
	SplitTransferSetting splitTransferSettingResponse `json:"split_transfer_setting"`
}

type networkResponse struct {
	ID            string                `json:"id"`
	Command       commandType           `json:"command"`
	KeepAlive     keepAliveResponse     `json:"keep_alive"`
	Download      downloadResponse      `json:"download"`
	Upload        uploadResponse        `json:"upload"`
	Progress      progressResponse      `json:"progress"`
	SplitTransfer splitTransferResponse `json:"splitTransfer"`
	Settings      settingsResponse      `json:"settings"`
	Error         error                 `json:"error"`
}
