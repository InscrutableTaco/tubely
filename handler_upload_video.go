package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {

	// set a limit on the size of the upload
	const maxMemory = 1 << 30

	// get the id of the video the upload is for
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid video ID", err)
		return
	}

	// authenticate the user
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	// retrieve video record from the database
	video, err := cfg.db.GetVideo(videoID)
	if errors.Is(err, sql.ErrNoRows) {
		respondWithError(w, http.StatusNotFound, "Video not found", err)
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't retrieve video", err)
		return
	}

	// check if user is authorized
	if video.UserID != userID {
		respondWithError(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	// log that we are starting the upload
	fmt.Println("uploading video", videoID, "by user", userID)

	// parse the request body
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse request", err)
		return
	}

	// store file / header in memory
	multipartFile, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't read file/headers", err)
		return
	}
	defer multipartFile.Close()

	// determine content Type for extension
	rawContentType := header.Header.Get("Content-Type")
	contentType, _, err := mime.ParseMediaType(rawContentType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't read file content-type", err)
		return
	}
	if contentType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid content type", nil)
		return
	}

	// create temporary file
	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create video file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// copy from multipart file to temporary file
	io.Copy(tempFile, multipartFile)
	tempFile.Seek(0, io.SeekStart)

	// derive 'folder' from aspect ratio
	aspectRatio, err := getVideoAspectRatio(tempFile.Name())
	if err != nil {
		log.Printf("Failed to obtain aspect ratio: %s", err.Error())
	}
	var folder string
	switch aspectRatio {
	case "16:9":
		folder = "landscape/"
	case "9:16":
		folder = "portrait/"
	default:
		folder = "other/"
	}

	// create a randomized string for the file name to prevent caching
	randomBytes := make([]byte, 8)
	_, err = rand.Read(randomBytes)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create random string for file name", err)
		return
	}
	randomString := base64.RawURLEncoding.EncodeToString(randomBytes)
	key := folder + randomString + ".mp4"

	// process video for fast start
	processedPath, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't process video for fast start", err)
		return
	}

	// open processed video
	uploadFile, err := os.Open(processedPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't open processed file for upload", err)
		return
	}
	defer uploadFile.Close()
	defer os.Remove(processedPath)

	// put video in the bucket
	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      aws.String(cfg.s3Bucket),
		Key:         aws.String(key),
		Body:        uploadFile,
		ContentType: &contentType,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to upload to S3", err)
		return
	}

	// update video record with the video url

	//videoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, key) // old url
	videoURL := fmt.Sprintf("%s,%s", cfg.s3Bucket, key) // new url
	video.VideoURL = &videoURL

	log.Printf("about to update id=%s url=%q", video.ID, videoURL)
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}
	v2, _ := cfg.db.GetVideo(video.ID)
	log.Printf("after update id=%s url=%v", v2.ID, v2.VideoURL)

	signedVideo, err := cfg.dbVideoToSignedVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating signed url", err)
		return
	}

	// success response
	respondWithJSON(w, http.StatusOK, signedVideo)

}
