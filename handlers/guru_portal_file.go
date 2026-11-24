package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// guruDaftarFile membaca format baru (JSON array) dan URL tunggal yang tersimpan lama.
func guruDaftarFile(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	var files []string
	if json.Unmarshal([]byte(raw), &files) == nil && files != nil {
		return files
	}
	return []string{raw}
}

func guruSimpanBanyakFile(c *gin.Context, field, sub string, ekstensi []string, maksMB int64) ([]string, bool, error) {
	form, err := c.MultipartForm()
	if err != nil {
		if errors.Is(err, http.ErrNotMultipart) {
			return nil, false, nil
		}
		return nil, false, errors.New("form upload tidak valid")
	}
	headers := form.File[field]
	if len(headers) == 0 {
		headers = form.File["lampiran"]
	}
	if len(headers) == 0 {
		if fh, err := c.FormFile("file"); err == nil {
			headers = []*multipart.FileHeader{fh}
		}
	}
	if len(headers) == 0 {
		return nil, false, nil
	}
	if len(headers) > 20 {
		return nil, false, fmt.Errorf("maksimal 20 file dalam satu kiriman")
	}
	urls := make([]string, 0, len(headers))
	for _, fh := range headers {
		ext := strings.ToLower(filepath.Ext(fh.Filename))
		allowed := false
		for _, candidate := range ekstensi {
			if candidate == ext {
				allowed = true
				break
			}
		}
		if !allowed || fh.Size > maksMB*1024*1024 {
			for _, saved := range urls {
				guruHapusFile(saved)
			}
			if !allowed {
				return nil, false, fmt.Errorf("format file %s tidak didukung", filepath.Base(fh.Filename))
			}
			return nil, false, fmt.Errorf("ukuran setiap file maksimal %d MB", maksMB)
		}
		dir := filepath.Join("uploads", sub)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			for _, saved := range urls {
				guruHapusFile(saved)
			}
			return nil, false, err
		}
		base := guruNamaFileAman.ReplaceAllString(strings.TrimSuffix(filepath.Base(fh.Filename), ext), "_")
		if base == "" {
			base = "file"
		}
		name := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), base, ext)
		if err := c.SaveUploadedFile(fh, filepath.Join(dir, name)); err != nil {
			for _, saved := range urls {
				guruHapusFile(saved)
			}
			return nil, false, err
		}
		urls = append(urls, "/uploads/"+sub+"/"+name)
	}
	return urls, true, nil
}

func guruJSONDaftarFile(raw string) []gin.H {
	urls := guruDaftarFile(raw)
	out := make([]gin.H, 0, len(urls))
	for _, fileURL := range urls {
		out = append(out, gin.H{"url": fileURL, "nama": guruNamaTampil(fileURL)})
	}
	return out
}

func guruSimpanDaftarFile(files []string) string {
	data, _ := json.Marshal(files)
	return string(data)
}

func guruFileIndex(c *gin.Context, raw string) string {
	files := guruDaftarFile(raw)
	index := guruAtoiUint(c.Query("index"))
	if len(files) == 0 || int(index) >= len(files) {
		return ""
	}
	return files[index]
}

var (
	guruNamaFileAman = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
	guruAwalanAngka  = regexp.MustCompile(`^\d+_`)
)

func guruAtoiUint(s string) uint {
	n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return uint(n)
}

// guruSimpanFile menyimpan upload ke ./uploads/<sub>/ dan mengembalikan URL publik.
// Hasil: (url, adaFile, error). Kalau tidak ada file, adaFile = false tanpa error.
func guruSimpanFile(c *gin.Context, field, sub string, ekstensi []string, maksMB int64) (string, bool, error) {
	fh, err := c.FormFile(field)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) || errors.Is(err, http.ErrNotMultipart) {
			return "", false, nil
		}
		return "", false, errors.New("upload file gagal dibaca")
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	diizinkan := false
	for _, e := range ekstensi {
		if e == ext {
			diizinkan = true
		}
	}
	if !diizinkan {
		return "", false, fmt.Errorf("format file tidak didukung (boleh: %s)", strings.Join(ekstensi, ", "))
	}
	if fh.Size > maksMB*1024*1024 {
		return "", false, fmt.Errorf("ukuran file maksimal %d MB", maksMB)
	}
	dir := filepath.Join("uploads", sub)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", false, err
	}
	dasar := guruNamaFileAman.ReplaceAllString(strings.TrimSuffix(filepath.Base(fh.Filename), filepath.Ext(fh.Filename)), "_")
	if dasar == "" {
		dasar = "file"
	}
	nama := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), dasar, ext)
	if err := c.SaveUploadedFile(fh, filepath.Join(dir, nama)); err != nil {
		return "", false, err
	}
	return "/uploads/" + sub + "/" + nama, true, nil
}

// guruPathFile mengubah URL/path di database menjadi path aman di dalam folder uploads.
func guruPathFile(fileURL string) string {
	p := strings.TrimSpace(fileURL)
	if p == "" {
		return ""
	}
	if u, err := url.Parse(p); err == nil && u.Path != "" {
		p = u.Path
	}
	p = strings.TrimPrefix(filepath.ToSlash(p), "/")
	if !strings.HasPrefix(p, "uploads/") {
		p = "uploads/" + p
	}
	bersih := filepath.Clean(filepath.FromSlash(p))
	if !strings.HasPrefix(bersih, "uploads"+string(os.PathSeparator)) {
		return ""
	}
	return bersih
}

func guruFileAda(fileURL string) bool {
	p := guruPathFile(fileURL)
	if p == "" {
		return false
	}
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func guruNamaTampil(fileURL string) string {
	if strings.TrimSpace(fileURL) == "" {
		return ""
	}
	nama := path.Base(strings.ReplaceAll(fileURL, "\\", "/"))
	return guruAwalanAngka.ReplaceAllString(nama, "")
}

func guruHapusFile(fileURL string) {
	p := guruPathFile(fileURL)
	if p == "" {
		return
	}
	slash := filepath.ToSlash(p)
	if strings.HasPrefix(slash, "uploads/materi/") || strings.HasPrefix(slash, "uploads/tugas/") {
		_ = os.Remove(p)
	}
}

func guruKirimFile(c *gin.Context, fileURL string) {
	p := guruPathFile(fileURL)
	if p == "" {
		guruPortalGagal(c, http.StatusNotFound, "File tidak tersedia")
		return
	}
	info, err := os.Stat(p)
	if err != nil || info.IsDir() {
		guruPortalGagal(c, http.StatusNotFound, "File tidak ditemukan di server")
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%q", guruNamaTampil(fileURL)))
	c.File(p)
}
