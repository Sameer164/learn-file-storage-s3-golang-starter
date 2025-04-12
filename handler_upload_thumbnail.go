package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

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
		respondWithError(w, http.StatusBadRequest, "Couldn't parse multipart form", err)
		return
	}
	file, header, err := r.FormFile("thumbnail")
	defer file.Close()
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse file", err)
		return
	}
	contentType := header.Header.Get("Content-Type")
	t, _, err := mime.ParseMediaType(contentType)
	if !(t == "image/jpeg" || t == "image/png") {
		respondWithError(w, http.StatusBadRequest, "Invalid content type", err)
		return
	}
	extension := getExtension(contentType)
	if extension == "" {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse file", err)
		return
	}
	randomName := make([]byte, 32)
	_, err = rand.Read(randomName)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't generate random name", err)
		return
	}
	fileName := base64.RawURLEncoding.EncodeToString(randomName)
	filePath := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%s.%s", fileName, extension))
	dataURL := fmt.Sprintf("http://localhost:%s/%s", cfg.port, filePath)

	createdFile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't create file", err)
		return
	}
	defer createdFile.Close()
	_, err = io.Copy(createdFile, file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't create file", err)
		return
	}
	v, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't find video", err)
		return
	}
	if v.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You are not authorized to upload this video", err)
		return
	}
	videoMetadata := database.Video{
		ID:           videoID,
		CreatedAt:    v.CreatedAt,
		UpdatedAt:    time.Now(),
		ThumbnailURL: &dataURL,
		VideoURL:     v.VideoURL,
		CreateVideoParams: database.CreateVideoParams{
			Title:       v.Title,
			Description: v.Description,
			UserID:      v.UserID,
		},
	}
	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}
	respondWithJSON(w, http.StatusOK, videoMetadata)
}
