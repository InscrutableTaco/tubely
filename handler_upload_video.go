package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	// get the id of the video the upload is for
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
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

	// log that we are starting the upload
	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// set a limit on the size of the upload
	const maxMemory = 10 << 20

	// parse the request body
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse request", err)
		return
	}

	// store file / header in memory
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't read file/headers", err)
		return
	}
	defer file.Close()

	// determine content Type for extension
	rawContentType := header.Header.Get("Content-Type")
	contentType, _, err := mime.ParseMediaType(rawContentType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't read file content-type", err)
		return
	}
	if contentType != "image/jpeg" && contentType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Invalid content type", nil)
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

	// check if user is authroized
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	// determine file extension
	var ext string
	typeSlice := strings.Split(contentType, "/")
	if len(typeSlice) > 1 {
		ext = typeSlice[len(typeSlice)-1]
	} else {
		ext = "png"
	}

	// create a randomized string for the file name to prevent caching
	randomBytes := make([]byte, 8)
	_, err = rand.Read(randomBytes)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create random string for file name", err)
		return
	}
	randomString := base64.RawURLEncoding.EncodeToString(randomBytes)

	// create the file
	filename := fmt.Sprintf("%s.%s", randomString, ext)
	filePath := filepath.Join(cfg.assetsRoot, filename)
	fileptr, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create thumbnail file", err)
		return
	}
	defer fileptr.Close()

	// copy the data to the file
	_, err = io.Copy(fileptr, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't copy thumbnail file", err)
		return
	}

	// update video record with the thumbnail url
	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, filename)
	video.ThumbnailURL = &thumbnailURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	// send a response cuz we done
	respondWithJSON(w, http.StatusOK, video)

}
