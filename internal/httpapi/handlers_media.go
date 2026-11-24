package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
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

var mediaMIME = map[string]struct {
	kind string
	ext  string
}{
	"image/jpeg":      {"image", ".jpg"},
	"image/png":       {"image", ".png"},
	"image/gif":       {"image", ".gif"},
	"image/webp":      {"image", ".webp"},
	"audio/mpeg":      {"audio", ".mp3"},
	"audio/mp4":       {"audio", ".m4a"},
	"audio/aac":       {"audio", ".aac"},
	"audio/wav":       {"audio", ".wav"},
	"audio/x-wav":     {"audio", ".wav"},
	"audio/ogg":       {"audio", ".ogg"},
	"video/mp4":       {"video", ".mp4"},
	"video/webm":      {"video", ".webm"},
	"video/quicktime": {"video", ".mov"},
}

var extMIME = map[string]string{
	".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png", ".gif": "image/gif", ".webp": "image/webp",
	".mp3": "audio/mpeg", ".m4a": "audio/mp4", ".aac": "audio/aac", ".wav": "audio/wav", ".ogg": "audio/ogg", ".oga": "audio/ogg",
	".mp4": "video/mp4", ".webm": "video/webm", ".mov": "video/quicktime",
}

func (s *Server) handleMediaUpload(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	task, _ := s.loadTaskFor(r, w)
	if task == nil {
		return
	}
	if !canReadTask(u, task) {
		writeError(w, notFound("待办不存在"))
		return
	}
	if !canWriteTask(u, task) {
		writeError(w, forbidden("没有为该待办添加附件的权限"))
		return
	}
	if task.Status == "completed" || task.ArchivedAt != nil {
		writeError(w, badRequest("已完成或已归档的待办不能添加附件"))
		return
	}
	limit := s.cfg.MaxVideoBytes
	if r.ContentLength > limit {
		writeError(w, &APIError{Status: http.StatusRequestEntityTooLarge, Code: "media.too_large", Message: "文件超出大小限制"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit+1<<20)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeError(w, &APIError{Status: http.StatusRequestEntityTooLarge, Code: "media.too_large", Message: "文件超出大小限制"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, badRequest("请选择要上传的文件"))
		return
	}
	defer file.Close()
	if header.Size <= 0 {
		writeError(w, badRequest("文件为空"))
		return
	}
	mimeType, kind, ok := detectMedia(file, header.Filename)
	if !ok {
		writeError(w, &APIError{Status: http.StatusUnsupportedMediaType, Code: "media.unsupported_type", Message: "不支持的文件类型"})
		return
	}
	maxBytes := s.cfg.MaxImageBytes
	if kind == "audio" {
		maxBytes = s.cfg.MaxAudioBytes
	} else if kind == "video" {
		maxBytes = s.cfg.MaxVideoBytes
	}
	if header.Size > maxBytes {
		writeError(w, &APIError{Status: http.StatusRequestEntityTooLarge, Code: "media.too_large", Message: "文件超出大小限制"})
		return
	}
	total, err := s.store.MediaTotalBytes(r.Context())
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	if total+header.Size > s.cfg.HouseholdQuota {
		writeError(w, &APIError{Status: http.StatusRequestEntityTooLarge, Code: "media.too_large", Message: "家庭存储空间已满"})
		return
	}
	occurrence := task.OccurrenceDate
	if v := strings.TrimSpace(r.FormValue("occurrenceDate")); v != "" {
		if err := checkDate(v); err != nil {
			writeError(w, err)
			return
		}
		occurrence = &v
	}
	count, err := s.store.CountMedia(r.Context(), task.ID, occurrence)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	if count >= 20 {
		writeError(w, badRequest("每个待办最多添加 20 个附件"))
		return
	}
	key, err := auth.RandomToken(16)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	key += mediaMIME[mimeType].ext
	if err := os.MkdirAll(filepath.Join(s.cfg.MediaDir(), "objects"), 0o750); err != nil {
		writeError(w, serverError(err))
		return
	}
	dst, err := os.OpenFile(s.mediaPath(key), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	written, copyErr := io.Copy(dst, file)
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(s.mediaPath(key))
		writeError(w, serverError(copyErr))
		return
	}
	name := filepath.Base(header.Filename)
	media := &model.TaskMedia{
		TaskID:         task.ID,
		OccurrenceDate: occurrence,
		StorageKey:     key,
		OriginalName:   &name,
		MediaType:      kind,
		MimeType:       mimeType,
		SizeBytes:      written,
		CreatedBy:      u.ID,
	}
	id, err := s.store.CreateMedia(r.Context(), media)
	if err != nil {
		_ = os.Remove(s.mediaPath(key))
		writeError(w, serverError(err))
		return
	}
	media.ID = id
	if kind == "image" {
		_ = s.makeThumbnail(media)
	}
	writeJSON(w, http.StatusCreated, map[string]any{"media": media})
}

func (s *Server) handleMediaGet(w http.ResponseWriter, r *http.Request, thumbnail bool) {
	u, _ := currentUser(r)
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	media, err := s.store.GetMedia(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, notFound("附件不存在"))
			return
		}
		writeError(w, serverError(err))
		return
	}
	task, err := s.store.GetTask(r.Context(), media.TaskID)
	if err != nil || !canReadTask(u, task) {
		writeError(w, notFound("附件不存在"))
		return
	}
	path := s.mediaPath(media.StorageKey)
	if thumbnail {
		if media.MediaType != "image" {
			writeError(w, notFound("缩略图不存在"))
			return
		}
		path = s.thumbPath(media.StorageKey)
		if _, err := os.Stat(path); err != nil {
			if err := s.makeThumbnail(media); err != nil {
				writeError(w, notFound("缩略图不存在"))
				return
			}
		}
		w.Header().Set("Content-Type", "image/jpeg")
	} else {
		w.Header().Set("Content-Type", media.MimeType)
		if media.OriginalName != nil {
			w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": *media.OriginalName}))
		}
	}
	w.Header().Set("Cache-Control", "private, max-age=86400")
	file, err := os.Open(path)
	if err != nil {
		writeError(w, notFound("附件文件不存在"))
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	http.ServeContent(w, r, media.StorageKey, info.ModTime(), file)
}

func (s *Server) handleMediaDelete(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	media, err := s.store.GetMedia(r.Context(), id)
	if err != nil {
		writeError(w, notFound("附件不存在"))
		return
	}
	task, err := s.store.GetTask(r.Context(), media.TaskID)
	if err != nil {
		writeError(w, notFound("附件不存在"))
		return
	}
	if !canReadTask(u, task) {
		writeError(w, notFound("附件不存在"))
		return
	}
	if !canWriteTask(u, task) {
		writeError(w, forbidden("没有删除该附件的权限"))
		return
	}
	if err := s.store.DeleteMedia(r.Context(), id); err != nil {
		writeError(w, serverError(err))
		return
	}
	if err := os.Remove(s.mediaPath(media.StorageKey)); err != nil && !os.IsNotExist(err) {
		writeError(w, serverError(err))
		return
	}
	_ = os.Remove(s.thumbPath(media.StorageKey))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleTaskMediaList(w http.ResponseWriter, r *http.Request) {
	task, _ := s.loadTaskFor(r, w)
	if task == nil {
		return
	}
	u, _ := currentUser(r)
	if !canReadTask(u, task) {
		writeError(w, notFound("待办不存在"))
		return
	}
	occurrence := task.OccurrenceDate
	if v := strings.TrimSpace(r.URL.Query().Get("occurrenceDate")); v != "" {
		occurrence = &v
	}
	items, err := s.store.ListMedia(r.Context(), task.ID, occurrence)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) mediaPath(key string) string {
	return filepath.Join(s.cfg.MediaDir(), "objects", key)
}

func (s *Server) thumbPath(key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(s.cfg.MediaDir(), "thumbs", hex.EncodeToString(sum[:16])+".jpg")
}

func (s *Server) makeThumbnail(m *model.TaskMedia) error {
	src, err := os.Open(s.mediaPath(m.StorageKey))
	if err != nil {
		return err
	}
	defer src.Close()
	img, _, err := image.Decode(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(s.cfg.MediaDir(), "thumbs"), 0o750); err != nil {
		return err
	}
	scaled := scaleToWidth(img, 480)
	tmp := s.thumbPath(m.StorageKey) + ".tmp"
	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}
	if err := jpeg.Encode(out, scaled, &jpeg.Options{Quality: 80}); err != nil {
		out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, s.thumbPath(m.StorageKey))
}

func scaleToWidth(src image.Image, maxWidth int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxWidth {
		return src
	}
	nw := maxWidth
	nh := h * maxWidth / w
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		sy := b.Min.Y + y*h/nh
		for x := 0; x < nw; x++ {
			sx := b.Min.X + x*w/nw
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

func (s *Server) removeMediaFiles(items []model.TaskMedia) {
	for _, m := range items {
		_ = os.Remove(s.mediaPath(m.StorageKey))
		_ = os.Remove(s.thumbPath(m.StorageKey))
	}
}

func detectMedia(file interface {
	Read([]byte) (int, error)
	Seek(int64, int) (int64, error)
}, filename string) (string, string, bool) {
	head := make([]byte, 512)
	n, _ := io.ReadFull(file, head)
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", "", false
	}
	sniffed := http.DetectContentType(head[:n])
	if _, ok := mediaMIME[sniffed]; ok {
		info := mediaMIME[sniffed]
		return sniffed, info.kind, true
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if mimeType, ok := extMIME[ext]; ok {
		info := mediaMIME[mimeType]
		return mimeType, info.kind, true
	}
	return "", "", false
}
