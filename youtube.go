package main

import (
	"errors"
	"fmt"
	"mime"
	"os/exec"
	"time"

	"github.com/kkdai/youtube/v2"
)

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

func durationFormat(seconds float64) string {
	var t time.Time
	t = t.Add(time.Duration(seconds) * time.Second)
	return t.Format("15:04:05")
}

func checkFFmpeg() (string, bool) {
	for _, ffmpeg := range []string{"ffmpeg", "bin/ffmpeg", "./bin/ffmpeg"} {
		if err := exec.Command(ffmpeg, "-version").Run(); err == nil {
			return ffmpeg, true
		}
	}
	return "", false
}
