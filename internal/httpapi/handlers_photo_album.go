package httpapi

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"mymoments/internal/auth"
	"mymoments/internal/model"
	"mymoments/internal/store"
)

const photoAlbumMaxCount = 20

var photoAlbumImageMIME = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

func (s *Server) photoAlbumPath(key string) string {
	return filepath.Join(s.cfg.PhotoAlbumDir(), filepath.Base(key))
}

// handlePhotoAlbumUpload 上传一张照片到首页照片墙。
func (s *Server) handlePhotoAlbumUpload(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	limit := s.cfg.MaxImageBytes
	if r.ContentLength > limit {
		writeError(w, &APIError{Status: http.StatusRequestEntityTooLarge, Code: "media.too_large", Message: "图片超出大小限制"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit+1<<20)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeError(w, &APIError{Status: http.StatusRequestEntityTooLarge, Code: "media.too_large", Message: "图片超出大小限制"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, badRequest("请选择要上传的图片"))
		return
	}
	defer file.Close()
	if header.Size <= 0 {
		writeError(w, badRequest("图片为空"))
		return
	}
	mimeType, kind, ok := detectMedia(file, header.Filename)
	if !ok || kind != "image" || !photoAlbumImageMIME[mimeType] {
		writeError(w, &APIError{Status: http.StatusUnsupportedMediaType, Code: "media.unsupported_type", Message: "仅支持 png、webp、jpg 格式的图片"})
		return
	}
	if header.Size > limit {
		writeError(w, &APIError{Status: http.StatusRequestEntityTooLarge, Code: "media.too_large", Message: "图片超出大小限制"})
		return
	}
	fit := strings.TrimSpace(r.FormValue("fit"))
	if fit == "" {
		fit = "cover"
	}
	if fit != "cover" && fit != "contain" {
		writeError(w, badRequest("显示方式无效"))
		return
	}
	count, err := s.store.CountPhotos(r.Context())
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	if count >= photoAlbumMaxCount {
		writeError(w, badRequest("照片墙最多添加 20 张照片"))
		return
	}
	key, err := auth.RandomToken(16)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	key += mediaMIME[mimeType].ext
	if err := os.MkdirAll(s.cfg.PhotoAlbumDir(), 0o750); err != nil {
		writeError(w, serverError(err))
		return
	}
	dst, err := os.OpenFile(s.photoAlbumPath(key), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	written, copyErr := io.Copy(dst, file)
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(s.photoAlbumPath(key))
		if copyErr != nil {
			writeError(w, serverError(copyErr))
		} else {
			writeError(w, serverError(closeErr))
		}
		return
	}
	name := filepath.Base(header.Filename)
	photo := &model.PhotoAlbum{
		StorageKey:   key,
		OriginalName: &name,
		MimeType:     mimeType,
		SizeBytes:    written,
		Fit:          fit,
		CreatedBy:    u.ID,
	}
	id, err := s.store.CreatePhoto(r.Context(), photo)
	if err != nil {
		_ = os.Remove(s.photoAlbumPath(key))
		writeError(w, serverError(err))
		return
	}
	photo.ID = id
	writeJSON(w, http.StatusCreated, map[string]any{"photo": photo})
}

// handlePhotoAlbumList 返回照片墙全部照片。
func (s *Server) handlePhotoAlbumList(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListPhotos(r.Context())
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// handlePhotoAlbumGet 读取照片墙图片文件。
func (s *Server) handlePhotoAlbumGet(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	photo, err := s.store.GetPhoto(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, notFound("照片不存在"))
			return
		}
		writeError(w, serverError(err))
		return
	}
	path := s.photoAlbumPath(photo.StorageKey)
	file, err := os.Open(path)
	if err != nil {
		writeError(w, notFound("照片文件不存在"))
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	w.Header().Set("Content-Type", photo.MimeType)
	if photo.OriginalName != nil {
		w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": *photo.OriginalName}))
	}
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeContent(w, r, photo.StorageKey, info.ModTime(), file)
}

// handlePhotoAlbumDelete 删除照片墙中的一张照片。
func (s *Server) handlePhotoAlbumDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	photo, err := s.store.GetPhoto(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, notFound("照片不存在"))
			return
		}
		writeError(w, serverError(err))
		return
	}
	if err := s.store.DeletePhoto(r.Context(), id); err != nil {
		writeError(w, serverError(err))
		return
	}
	_ = os.Remove(s.photoAlbumPath(photo.StorageKey))
	w.WriteHeader(http.StatusNoContent)
}
