package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"os/exec"
	"strings"
)

func getExtension(formType string) string {
	if formType == "" {
		return ""
	}
	data := strings.Split(formType, "/")
	if len(data) == 2 {
		return data[1]
	}
	return ""
}

func getVideoAspectRatio(filePath string) (string, error) {
	buf := &bytes.Buffer{}
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-print_format", "json",
		"-show_streams",
		filePath, // File path as separate argument
	)
	cmd.Stdout = buf
	if err := cmd.Run(); err != nil {
		return "", err
	}
	stream := make(map[string]interface{})
	err := json.Unmarshal(buf.Bytes(), &stream)
	if err != nil {
		return "", err
	}

	if _, ok := stream["streams"]; !ok {
		return "", errors.New("no streams found")
	}
	streams := stream["streams"].([]interface{})
	info := streams[0].(map[string]interface{})
	width := info["width"].(float64)
	height := info["height"].(float64)

	ratio := width / height

	// Define common ratios with tolerance
	const tolerance = 0.05 // 5% tolerance for floating point comparison
	commonRatios := map[string]float64{
		"16:9": 16.0 / 9.0,
		"9:16": 9.0 / 16.0,
	}

	// Check against common ratios with tolerance
	for name, targetRatio := range commonRatios {
		if math.Abs(ratio-targetRatio) < tolerance {
			return name, nil
		}
	}
	return "other", nil
}

func processVideoForFastStart(filePath string) (string, error) {
	outputPath := filePath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputPath)
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return outputPath, nil
}
