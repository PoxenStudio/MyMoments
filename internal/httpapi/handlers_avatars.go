package httpapi

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"mymoments/internal/auth"
	"mymoments/internal/model"
)

var avatarKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,96}\.[A-Za-z0-9]{2,5}$`)

func (s *Server) avatarDir() string {
	return filepath.Join(s.cfg.MediaDir(), "avatars")
}

func (s *Server) avatarPath(key string) string {
	return filepath.Join(s.avatarDir(), filepath.Base(key))
}

// handleAvatarUpload 上传当前用户头像，返回可在成员资料中引用的 key。
func (s *Server) handleAvatarUpload(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	s.saveAvatar(w, r, u)
}

// handleMemberAvatarUpload 管理员为指定成员上传头像。
func (s *Server) handleMemberAvatarUpload(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	u, err := s.store.GetUserByID(r.Context(), id)
	if err != nil {
		writeError(w, notFound("成员不存在"))
		return
	}
	s.saveAvatar(w, r, u)
}

func (s *Server) saveAvatar(w http.ResponseWriter, r *http.Request, u *model.User) {
	limit := s.cfg.MaxImageBytes
	if r.ContentLength > limit {
		writeError(w, &APIError{Status: http.StatusRequestEntityTooLarge, Code: "media.too_large", Message: "头像文件超出大小限制"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit+1<<20)
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		writeError(w, &APIError{Status: http.StatusRequestEntityTooLarge, Code: "media.too_large", Message: "头像文件超出大小限制"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, badRequest("请选择要上传的头像"))
		return
	}
	defer file.Close()
	if header.Size <= 0 {
		writeError(w, badRequest("头像文件为空"))
		return
	}
	mimeType, kind, ok := detectMedia(file, header.Filename)
	if !ok || kind != "image" {
		writeError(w, &APIError{Status: http.StatusUnsupportedMediaType, Code: "media.unsupported_type", Message: "头像仅支持图片格式"})
		return
	}
	if header.Size > limit {
		writeError(w, &APIError{Status: http.StatusRequestEntityTooLarge, Code: "media.too_large", Message: "头像文件超出大小限制"})
		return
	}
	key, err := auth.RandomToken(16)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	key += mediaMIME[mimeType].ext
	if err := os.MkdirAll(s.avatarDir(), 0o750); err != nil {
		writeError(w, serverError(err))
		return
	}
	dst, err := os.OpenFile(s.avatarPath(key), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	written, copyErr := io.Copy(dst, file)
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil || written == 0 {
		_ = os.Remove(s.avatarPath(key))
		writeError(w, serverError(copyErr))
		return
	}
	old := u.AvatarKey
	u.AvatarKey = &key
	if err := s.store.UpdateUser(r.Context(), u); err != nil {
		_ = os.Remove(s.avatarPath(key))
		writeError(w, serverError(err))
		return
	}
	if old != nil && *old != key && avatarKeyPattern.MatchString(*old) {
		_ = os.Remove(s.avatarPath(*old))
	}
	writeJSON(w, http.StatusOK, map[string]any{"avatarKey": key})
}

// handleAvatarGet 读取家庭成员头像。
func (s *Server) handleAvatarGet(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.PathValue("key"))
	if !avatarKeyPattern.MatchString(key) {
		writeError(w, notFound("头像不存在"))
		return
	}
	file, err := os.Open(s.avatarPath(key))
	if err != nil {
		writeError(w, notFound("头像不存在"))
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	if ct, ok := extMIME[strings.ToLower(filepath.Ext(key))]; ok {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeContent(w, r, key, info.ModTime(), file)
}
