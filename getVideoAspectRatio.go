package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {

	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	// Define struct to match ffprobe output
	type Stream struct {
		Width  int    `json:"width"`
		Height int    `json:"height"`
		Codec  string `json:"codec_type"`
	}
	type FFProbeOutput struct {
		Streams []Stream `json:"streams"`
	}

	var result FFProbeOutput

	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		return "", err
	}

	// Find the first video stream
	for _, stream := range result.Streams {
		if stream.Codec == "video" && stream.Width > 0 && stream.Height > 0 {
			var aspect string
			actualRatio := float64(stream.Width) / float64(stream.Height)

			const epsilon = 0.01 // A small tolerance value

			// Define target ratios
			targetLandscapeRatio := 16.0 / 9.0
			targetPortraitRatio := 9.0 / 16.0

			if math.Abs(actualRatio-targetLandscapeRatio) < epsilon {
				aspect = "16:9"
			} else if math.Abs(actualRatio-targetPortraitRatio) < epsilon {
				aspect = "9:16"
			} else {
				aspect = "other"
			}
			return aspect, nil
		}
	}

	return "", fmt.Errorf("no video stream found")

}
