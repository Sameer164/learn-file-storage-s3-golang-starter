package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
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
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)
	file, header, err := r.FormFile("video")
	defer file.Close()
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse file", err)
		return
	}
	t, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if t != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse file", err)
		return
	}
	createdFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't create file", err)
		return
	}
	defer os.Remove(createdFile.Name())
	defer createdFile.Close()

	_, err = io.Copy(createdFile, file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't create file", err)
		return
	}
	_, err = createdFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't seek file", err)
		return
	}

	aspectRatio, err := getVideoAspectRatio(createdFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't get aspect ratio", err)
		return
	}
	randomName := make([]byte, 32)
	_, err = rand.Read(randomName)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't generate random name", err)
		return
	}
	fileName := base64.RawURLEncoding.EncodeToString(randomName)
	fileName = fmt.Sprintf("%s.mp4", fileName)
	if aspectRatio == "16:9" {
		fileName = filepath.Join("landscape", fileName)
	} else if aspectRatio == "9:16" {
		fileName = filepath.Join("portrait", fileName)
	} else {
		fileName = filepath.Join("other", fileName)
	}

	processedVideo, err := processVideoForFastStart(createdFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't process video", err)
		return
	}
	fastVideo, err := os.OpenFile(processedVideo, os.O_RDWR, 0600)
	defer fastVideo.Close()
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't open file", err)
		return
	}
	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      aws.String(cfg.s3Bucket),
		Key:         aws.String(fileName),
		Body:        fastVideo,
		ContentType: aws.String("video/mp4"),
	})

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't upload video", err)
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

	fileURL := fmt.Sprintf("https://%s/%s", cfg.cloudfrontDomain, fileName)

	videoMetadata := database.Video{
		ID:           videoID,
		CreatedAt:    v.CreatedAt,
		UpdatedAt:    time.Now(),
		ThumbnailURL: v.ThumbnailURL,
		VideoURL:     &fileURL,
		CreateVideoParams: database.CreateVideoParams{
			Title:       v.Title,
			Description: v.Description,
			UserID:      v.UserID,
		},
	}

	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't save video", err)
		return
	}
	respondWithJSON(w, http.StatusOK, videoMetadata)
}
