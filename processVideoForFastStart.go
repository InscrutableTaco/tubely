package main

import "os/exec"

func processVideoForFastStart(filePath string) (string, error) {

	// create new string for output filepath
	outputPath := filePath + ".processing"

	// define command and parameters
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputPath)

	// run it
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	// return output filepath
	return outputPath, nil

}
