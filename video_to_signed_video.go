package main

import (
	"fmt"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"strings"
	"time"
)

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	bucketAndKey := strings.Split(*(video.VideoURL), ",")

	if len(bucketAndKey) != 2 {
		return database.Video{}, fmt.Errorf("Invalid bucket and key")
	}

	bucket := bucketAndKey[0]
	key := bucketAndKey[1]
	url, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Duration(5*time.Minute))
	fmt.Println("url:", url)
	if err != nil {
		return database.Video{}, err
	}
	video.VideoURL = &url
	return video, nil
}
