package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os/exec"
	"time"

	"github.com/yms2772/download_accelerator/cmd"

	"github.com/kkdai/youtube/v2"
)

type chapterData struct {
	Title       string `json:"title"`
	StartMillis int64  `json:"start_millis"`
}

type heatSeekerData struct {
	Score          float64 `json:"score"`
	StartMillis    int64   `json:"start_millis"`
	DurationMillis int64   `json:"duration_millis"`
}

type apiResponse struct {
	Title      string           `json:"title"`
	SubTitle   string           `json:"sub_title"`
	Chapter    []chapterData    `json:"chapter"`
	HeatSeeker []heatSeekerData `json:"heat_seeker"`
}

func youtubeQuality(formats youtube.FormatList) []string {
	var quality []string
	for _, format := range formats {
		mediaType, params, _ := mime.ParseMediaType(format.MimeType)
		quality = append(quality, fmt.Sprintf("[%s] %s - %s", mediaType, format.QualityLabel, params["codecs"]))
	}
	return quality
}

func youtubeAudio(formats youtube.FormatList) (youtube.Format, error) {
	for _, format := range formats {
		mediaType, _, _ := mime.ParseMediaType(format.MimeType)
		if mediaType == "audio/mp4" {
			return format, nil
		}
	}
	return youtube.Format{}, errors.New("cannot find audio")
}

func youtubeHeatSeeker(id string) (apiResponse, error) {
	log.Println(id)
	resp, err := http.Get("https://heatseeker.mokky.kr/api?v=" + id)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return apiResponse{}, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return apiResponse{}, err
	}

	var data apiResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return apiResponse{}, err
	}
	return data, nil
}

func durationFormat(seconds float64) string {
	var t time.Time
	t = t.Add(time.Duration(seconds) * time.Second)
	return t.Format("15:04:05")
}

func checkFFmpeg() (string, bool) {
	for _, ffmpeg := range []string{"ffmpeg", "bin/ffmpeg", "./bin/ffmpeg"} {
		if err := cmd.PrepareBackgroundCommand(exec.Command(ffmpeg, "-version")).Run(); err == nil {
			return ffmpeg, true
		}
	}
	return "", false
}
