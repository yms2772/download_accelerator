package main

import (
	"fmt"
	"mime"
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

func durationFormat(seconds float64) string {
	var t time.Time
	t = t.Add(time.Duration(seconds) * time.Second)
	return t.Format("15:04:05")
}
