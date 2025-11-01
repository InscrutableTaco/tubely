package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {

	// error if url absent
	if video.VideoURL == nil || *video.VideoURL == "" {
		return database.Video{}, errors.New("video url empty")
	}

	// parse the url
	raw := strings.TrimSpace(*video.VideoURL)
	parts := strings.Split(raw, ",")
	if len(parts) != 2 {
		return database.Video{}, fmt.Errorf("unable to parse video url for signing: %q", raw)
	}
	bucket := strings.TrimSpace(parts[0])
	key := strings.TrimSpace(parts[1])
	if bucket == "" || key == "" {
		return database.Video{}, fmt.Errorf("unable to parse video url for signing: bucket=%q key=%q", bucket, key)
	}

	// create a presigned url
	presignedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Minute*2)
	if err != nil {
		return database.Video{}, fmt.Errorf("failed to generate presigned url: %s", err)
	}

	// update and return video (video in db isn't modified)
	video.VideoURL = &presignedURL
	return video, nil

}
