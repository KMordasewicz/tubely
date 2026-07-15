package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"mime"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func getFileExtentionFromContentType(contentType string, allowedExtentions []string) (mediaType, extention string, err error) {
	mediaType, _, err = mime.ParseMediaType(contentType)
	if err != nil {
		return "", "", err
	}

	extensions, err := mime.ExtensionsByType(mediaType)
	if err != nil {
		return "", "", err
	}

	var fileExtensions []string
	for _, ext := range allowedExtentions {
		fileExtensions = append(fileExtensions, "."+ext)
	}

	for _, ext := range extensions {
		if slices.Contains(fileExtensions, ext) {
			return mediaType, ext, nil
		} else {
			log.Printf("Additional extension mapping matched: %s\n", ext)
		}
	}

	return "", "", fmt.Errorf("unsuported image format: %v, supported ones: %v", extensions, allowedExtentions)
}

func generateFileName(ext string) (string, error) {
	nameBytes := make([]byte, 32)
	_, err := rand.Read(nameBytes)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(nameBytes) + ext, nil
}

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignedClient := s3.NewPresignClient(s3Client)
	presignedRequest, err := presignedClient.PresignGetObject(
		context.Background(),
		&s3.GetObjectInput{
			Bucket: &bucket,
			Key:    &key,
		},
		s3.WithPresignExpires(expireTime),
	)
	if err != nil {
		return "", err
	}

	return presignedRequest.URL, nil
}
