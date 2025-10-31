package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {

	if video.VideoURL == nil {
		return database.Video{}, errors.New("video url empty")
	}

	if video.VideoURL == nil || *video.VideoURL == "" {
		return database.Video{}, errors.New("video url empty")
	}
	raw := strings.TrimSpace(*video.VideoURL)
	log.Printf("Signing raw video_url: %q", raw)
	parts := strings.Split(raw, ",")
	if len(parts) != 2 {
		return database.Video{}, fmt.Errorf("unable to parse video url for signing: %q", raw)
	}
	bucket := strings.TrimSpace(parts[0])
	key := strings.TrimSpace(parts[1])
	if bucket == "" || key == "" {
		return database.Video{}, fmt.Errorf("unable to parse video url for signing: bucket=%q key=%q", bucket, key)
	}

	presignedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Minute*2)
	if err != nil {
		return database.Video{}, fmt.Errorf("failed to generate presigned url: %s", err)
	}

	video.VideoURL = &presignedURL

	return video, nil

}
