package main

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {

	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)

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
		respondWithError(w, http.StatusBadRequest, "Unknown video id", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Access denied", err)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "File header missing", err)
		return
	}
	defer file.Close()

	mediaType := header.Header.Get("Content-Type")
	if len(mediaType) == 0 {
		respondWithError(w, http.StatusBadRequest, "Missing media type", err)
		return
	}
	if !isValidVideo(mediaType) {
		respondWithError(w, http.StatusUnsupportedMediaType, "Invalid media type", err)
		return
	}
	temp, err := os.CreateTemp("", "tubely-*.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to open video", err)
		return
	}
	defer os.Remove(temp.Name())
	defer temp.Close()
	_, err = io.Copy(temp, file)
	if err != nil {
		respondWithError(w, http.StatusInsufficientStorage, "Unable to save video", err)
		return
	}
	_, err = temp.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to rewind video", err)
		return
	}
	var keyBits [32]byte
	_, err = rand.Read(keyBits[:])
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Can't create stupid filename", err)
		return
	}
	key := base64.RawURLEncoding.EncodeToString(keyBits[:]) + ".mp4"

	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{Bucket: &cfg.s3Bucket, Key: &key, Body: temp, ContentType: &mediaType})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to store video", err)
		return
	}
	videoUrl := cfg.getS3Url() + key
	video.VideoURL = &videoUrl
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Can't update database", err)
		return
	}
	respondWithJSON(w, http.StatusOK, video)
}
