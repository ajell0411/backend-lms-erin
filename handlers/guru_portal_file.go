package handlers

import (
	"errors"
	"fmt"
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
