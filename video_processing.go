package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
)

type ContainerStream struct {
	Index     int    `json:"index"`
	CodecType string `json:"codec_type"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type Container struct {
	Streams []ContainerStream `json:"streams"`
}

type vidType struct {
	ratio    float64
	ratioStr string
	name     string
}

var vidTypes = []vidType{
	vidType{ratio: 16.0 / 9.0, ratioStr: "16:9", name: "landscape"},
	vidType{ratio: 9.0 / 16.0, ratioStr: "9:16", name: "portrait"},
}

func epsilonEq(a, b, epsilon float64) bool {
	return a >= (b-epsilon) && a <= (b+epsilon)
}

func getRatioStr(ratio float64) string {
	for _, rat := range vidTypes {
		if epsilonEq(ratio, rat.ratio, 0.1) {
			return rat.ratioStr
		}
	}
	return "other"
}

func getRatioName(ratioStr string) string {
	for _, rat := range vidTypes {
		if rat.ratioStr == ratioStr {
			return rat.name
		}
	}
	return "other"
}

func extractDim(c Container) (width, height int, err error) {

	for _, stream := range c.Streams {
		if stream.CodecType == "video" {
			if stream.Width == 0 || stream.Height == 0 {
				continue
			}
			return stream.Width, stream.Height, nil
		}
	}
	return 0, 0, fmt.Errorf("invalid video file")
}

func getVideoAspectRatio(filePath string) (string, error) {
	command := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	buffer := bytes.NewBuffer(make([]byte, 0))
	errbuf := bytes.NewBuffer(make([]byte, 0))
	command.Stdout = buffer
	command.Stderr = errbuf
	err := command.Run()
	if err != nil {
		log.Println(errbuf.String())
		return "", err
	}
	var vid Container
	err = json.Unmarshal(buffer.Bytes(), &vid)
	if err != nil {
		return "", err
	}
	width, height, err := extractDim(vid)
	if err != nil {
		return "", err
	}

	return getRatioStr(float64(width) / float64(height)), nil
}

func processVideoForFastStart(filePath string) (string, error) {

	processed := filePath + ".processed"
	command := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", processed)
	buffer := bytes.NewBuffer(make([]byte, 0))
	errbuf := bytes.NewBuffer(make([]byte, 0))
	command.Stdout = buffer
	command.Stderr = errbuf
	err := command.Run()
	if err != nil {
		log.Println(errbuf.String())
		return "", err
	}

	return processed, nil
}
