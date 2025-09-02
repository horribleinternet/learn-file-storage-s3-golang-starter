package main

import (
	"crypto/rand"
	"encoding/base64"
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

	//log.Println("uploading thumbnail for video ", videoID, " by user ", userID)

	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse form", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "File header missing", err)
		return
	}

	mediaType := header.Header.Get("Content-Type")
	if len(mediaType) == 0 {
		respondWithError(w, http.StatusBadRequest, "Missing media type", err)
		return
	}
	if !isValidThumbnail(mediaType) {
		respondWithError(w, http.StatusUnsupportedMediaType, "Invalid media type", err)
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
	//fmt.Println("header: ", mediaType, " ext: ", getFileExt(mediaType))
	var nameBytes [32]byte
	_, err = rand.Read(nameBytes[:])
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Can't create stupid filename", err)
		return
	}
	stupidName := base64.RawURLEncoding.EncodeToString(nameBytes[:])
	path := filepath.Join(cfg.assetsRoot, stupidName+"."+getFileExt(mediaType))
	thumbFile, err := os.Create(path)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Can't create thumbnail", err)
		return
	}
	_, err = io.Copy(thumbFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Can't save thumbnail", err)
		return
	}
	thumbUrl := cfg.getBaseURL() + path
	video.ThumbnailURL = &thumbUrl

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Can't update database", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
