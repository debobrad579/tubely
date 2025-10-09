package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/debobrad579/tubely/internal/auth"
	"github.com/google/uuid"
)

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-print_format", "json",
		"-show_streams",
		filePath,
	)

	var buf bytes.Buffer
	cmd.Stdout = &buf

	if err := cmd.Run(); err != nil {
		return "", err
	}

	type resultType struct {
		Streams []struct {
			Width  float64 `json:"width"`
			Height float64 `json:"height"`
		} `json:"streams"`
	}

	var result resultType
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		return "", err
	}

	if len(result.Streams) == 0 {
		return "", errors.New("No streams found in file")
	}

	ratio := result.Streams[0].Width / result.Streams[0].Height

	const landscape = 16.0 / 9.0
	const portrait = 9.0 / 16.0

	if ratio > landscape-0.05 && ratio < landscape+0.05 {
		return "landscape", nil
	} else if ratio > portrait-0.05 && ratio < portrait+0.05 {
		return "portrait", nil
	}

	return "other", nil
}

func processVideoForFastStart(filePath string) (string, error) {
	outputFilePath := filePath + ".processing"

	cmd := exec.Command("ffmpeg",
		"-i", filePath, "-c",
		"copy", "-movflags",
		"faststart", "-f",
		"mp4", outputFilePath,
	)

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return outputFilePath, nil
}

/*
func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignedClient := s3.NewPresignClient(s3Client)
	presignedURL, err := presignedClient.PresignGetObject(
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

	return presignedURL.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil
	}

	bucketKeySlice := strings.Split(*video.VideoURL, ",")
	if len(bucketKeySlice) != 2 {
		return video, errors.New("Incorrect video URL format")
	}

	presignedURL, err := generatePresignedURL(cfg.s3Client, bucketKeySlice[0], bucketKeySlice[1], 15*time.Second)
	if err != nil {
		return video, err
	}

	video.VideoURL = &presignedURL

	return video, nil
}
*/

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	r.Body = io.NopCloser(http.MaxBytesReader(w, r.Body, 1<<30))

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

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to get video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to form file", nil)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to parse media type", err)
		return
	}
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Incorrect media type", nil)
		return
	}

	tempFile, err := os.CreateTemp("", "tupely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create temp file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()
	if _, err := io.Copy(tempFile, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to copy file", err)
		return
	}

	tempFile.Seek(0, io.SeekStart)

	aspectRatio, err := getVideoAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get video aspect ratio", err)
		return
	}

	processedFilePath, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to process video", err)
		return
	}

	processedFile, err := os.Open(processedFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to read file", err)
		return
	}
	defer os.Remove(processedFilePath)
	defer processedFile.Close()

	fileExtention := strings.TrimPrefix(mediaType, "video/")
	b := make([]byte, 32)
	rand.Read(b)
	filename := base64.RawURLEncoding.EncodeToString(b)
	fileKey := fmt.Sprintf("%s/%s.%s", aspectRatio, filename, fileExtention)

	if _, err := cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{Bucket: &cfg.s3Bucket, Key: &fileKey, Body: processedFile, ContentType: &mediaType}); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to upload video", err)
		return
	}

	videoURL := fmt.Sprintf("https://%s/%s", cfg.s3CfDistribution, fileKey)
	video.VideoURL = &videoURL

	if err := cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
