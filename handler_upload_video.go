package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
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

	fmt.Println("uploading video", videoID, "by user", userID)

	r.Body = http.MaxBytesReader(w, r.Body, 1<<30) // 1GB

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video metadata", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not authorized to upload video", nil)
		return
	}

	err = r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error parsing multi part form", err)
		return
	}

	videoData, videoHeaderData, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't parse video data", err)
		return
	}
	defer videoData.Close()

	mediaType, ext, err := getFileExtentionFromContentType(videoHeaderData.Header.Get("Content-Type"), []string{"mp4"})
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Malformed content-type header value", err)
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't prepare video to save", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, videoData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error copying video data", err)
		return
	}

	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could't reset video ffile", err)
		return
	}

	aspectRatio, err := getVideoAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get file aspect ratio", err)
		return
	}

	fileName, err := generateFileName(ext)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't generate file name", err)
		return
	}
	fileName = path.Join(getVideoPrefix(aspectRatio), fileName)

	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      aws.String(cfg.s3Bucket),
		Key:         aws.String(fileName),
		Body:        tempFile,
		ContentType: aws.String(mediaType),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error uploading video to bucket", err)
		return
	}

	videoURL := cfg.getVideoURL(fileName)
	video.VideoURL = &videoURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could't update video metadata", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
