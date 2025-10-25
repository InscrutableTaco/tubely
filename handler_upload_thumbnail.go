package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	const maxMemory = 10 << 20

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse request", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't read file headers", err)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}

	video, err := cfg.db.GetVideo(videoID)
	if errors.Is(err, sql.ErrNoRows) {
		respondWithError(w, http.StatusNotFound, "Video not found", err)
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't retrieve video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	// Save the bytes to a file at the path /assets/<videoID>.<file_extension>
	var ext string
	typeSlice := strings.Split(contentType, "/")
	if len(typeSlice) > 1 {
		ext = typeSlice[len(typeSlice)-1]
	} else {
		ext = "png"
	}

	filename := fmt.Sprintf("%s.%s", videoIDString, ext)
	filePath := filepath.Join(cfg.assetsRoot, filename)
	fileptr, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create thumbnail file", err)
		return
	}
	defer fileptr.Close()

	_, err = io.Copy(fileptr, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't copy thumbnail file", err)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, filename)
	video.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)

}
