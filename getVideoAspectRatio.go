package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {

	// define the command to run ffprobe
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)

	// declare a variable to store the results in memory
	var out bytes.Buffer
	cmd.Stdout = &out

	// run the command
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	// define structs to match ffprobe output
	type Stream struct {
		Width  int    `json:"width"`
		Height int    `json:"height"`
		Codec  string `json:"codec_type"`
	}
	type FFProbeOutput struct {
		Streams []Stream `json:"streams"`
	}

	// prepare an empty slice of struct streams
	var result FFProbeOutput

	// unmarshal the json from the cmd output into the slice
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		return "", err
	}

	// find the first video stream
	for _, stream := range result.Streams {
		if stream.Codec == "video" && stream.Width > 0 && stream.Height > 0 {

			// calculate raw ratio
			actualRatio := float64(stream.Width) / float64(stream.Height)

			// Define target ratios
			targetLandscapeRatio := 16.0 / 9.0
			targetPortraitRatio := 9.0 / 16.0

			// A small tolerance value
			const epsilon = 0.01

			// compare raw ratio to target ratios and return appropriate string
			var aspect string
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

	// no video found, return error msg
	return "", fmt.Errorf("no video stream found")

}
