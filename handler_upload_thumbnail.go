package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

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

	err = r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error parsing multi part form", err)
		return
	}

	imageFileData, imageFileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error getting thumbnail file data", err)
		return
	}
	defer imageFileData.Close()

	mediaType := imageFileHeader.Header.Get("Content-Type")
	if mediaType == "" {
		respondWithError(w, http.StatusBadRequest, "Missing Content-Type for thumbnail", nil)
		return
	}

	_, ext, err := getFileExtentionFromContentType(mediaType, []string{"jpeg", "png"})
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Incorrect media type in content-type header", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error fetching video metadata", err)
		return
	}
	if userID != video.UserID {
		respondWithError(w, http.StatusUnauthorized, "Not the author of the video", nil)
		return
	}

	fileName, err := generateFileName(ext)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't generate thumbnail file name", err)
		return
	}
	filePath := filepath.Join(cfg.assetsRoot, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create thumbnail file", err)
		return
	}
	_, err = io.Copy(file, imageFileData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't save thumbnail", err)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%s/%s", cfg.port, filePath)
	video.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating video thumbnail", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
