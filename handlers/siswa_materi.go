package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
	"e-class-backend/models"
)

type MateriSiswaItem struct {
	ID        uint               `json:"id"`
	Judul     string             `json:"judul"`
	Guru      string             `json:"guru,omitempty"`
	NamaFile  string             `json:"nama_file"`
	Files     []LampiranInfo     `json:"files,omitempty"`
	Ekstensi  string             `json:"ekstensi,omitempty"`
	Ukuran    int64              `json:"ukuran,omitempty"`
	CreatedAt string             `json:"created_at"`
	Pelajaran PelajaranSiswaItem `json:"pelajaran"`
}

func formatMateriItem(m models.Materi) MateriSiswaItem {
	fileURLs := guruDaftarFile(m.FileUrl)
	firstURL := ""
	if len(fileURLs) > 0 {
		firstURL = fileURLs[0]
	}
	namaFile := filepath.Base(firstURL)
	if namaFile == "" || namaFile == "." || namaFile == "/" || !strings.Contains(namaFile, ".") {
		namaFile = fmt.Sprintf("%s.pdf", strings.ReplaceAll(m.Judul, "/", "-"))
	}
	ext := strings.TrimPrefix(filepath.Ext(namaFile), ".")
	if ext == "" {
		ext = "pdf"
	}

	var ukuran int64 = 1024 * 1024 // 1 MB default estimate
	localPath := filepath.Clean(firstURL)
	if strings.HasPrefix(localPath, "/") || strings.HasPrefix(localPath, "\\") {
		localPath = "." + localPath
	}
	if info, err := os.Stat(localPath); err == nil {
		ukuran = info.Size()
	}

	files := make([]LampiranInfo, 0, len(fileURLs))
	for _, fileURL := range fileURLs {
		files = append(files, LampiranInfo{Nama: guruNamaTampil(fileURL), Url: fileURL})
	}
	guruNama := strings.TrimSpace(m.Guru.Nama)
	if guruNama == "" && m.GuruID > 0 {
		var guru models.User
		if err := config.DB.Select("nama").First(&guru, m.GuruID).Error; err == nil {
			guruNama = strings.TrimSpace(guru.Nama)
		}
	}
	return MateriSiswaItem{
		ID:        m.ID,
		Judul:     m.Judul,
		Guru:      guruNama,
		NamaFile:  namaFile,
		Files:     files,
		Ekstensi:  ext,
		Ukuran:    ukuran,
		CreatedAt: m.CreatedAt.Format(time.RFC3339),
		Pelajaran: PelajaranSiswaItem{
			ID:   m.PelajaranID,
			Nama: m.Pelajaran.Nama,
			Kode: m.Pelajaran.Kode,
		},
	}
}

func ListSiswaMateri(c *gin.Context) {
	siswa, ok := getSiswaFromContext(c)
	if !ok {
		return
	}

	query := config.DB.Preload("Pelajaran").Preload("Guru").Where("kelas_id = ?", *siswa.KelasID)

	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(judul) LIKE ? OR LOWER(deskripsi) LIKE ?", like, like)
	}

	pelajaranIDStr := c.Query("pelajaran_id")
	if pelajaranIDStr == "" {
		pelajaranIDStr = c.Query("pelajaranId")
	}
	if pelajaranIDStr != "" {
		if pid, err := strconv.Atoi(pelajaranIDStr); err == nil && pid > 0 {
			query = query.Where("pelajaran_id = ?", pid)
		}
	}

	var materis []models.Materi
	if err := query.Order("created_at DESC").Order("id DESC").Find(&materis).Error; err != nil {
		sendSiswaError(c, http.StatusInternalServerError, "Gagal mengambil daftar materi")
		return
	}

	items := make([]MateriSiswaItem, 0, len(materis))
	for _, m := range materis {
		items = append(items, formatMateriItem(m))
	}

	sendSiswaSuccess(c, http.StatusOK, "Berhasil memuat daftar materi", items)
}

func UnduhSiswaMateri(c *gin.Context) {
	siswa, ok := getSiswaFromContext(c)
	if !ok {
		return
	}

	id := c.Param("id")
	var materi models.Materi
	if err := config.DB.Preload("Pelajaran").First(&materi, id).Error; err != nil {
		sendSiswaError(c, http.StatusNotFound, "Materi tidak ditemukan")
		return
	}

	if materi.KelasID != *siswa.KelasID {
		sendSiswaError(c, http.StatusForbidden, "Materi ini bukan untuk kelas Anda")
		return
	}

	fileURLs := guruDaftarFile(materi.FileUrl)
	index := guruAtoiUint(c.Query("index"))
	fileURL := ""
	if len(fileURLs) > 0 && int(index) >= len(fileURLs) {
		sendSiswaError(c, http.StatusNotFound, "File materi tidak ditemukan")
		return
	}
	if len(fileURLs) > 0 {
		fileURL = fileURLs[index]
	}
	namaFile := filepath.Base(fileURL)
	if namaFile == "" || namaFile == "." || namaFile == "/" || !strings.Contains(namaFile, ".") {
		namaFile = fmt.Sprintf("%s.pdf", strings.ReplaceAll(materi.Judul, "/", "-"))
	}

	encodedFilename := url.PathEscape(namaFile)
	disposition := fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", namaFile, encodedFilename)
	c.Header("Content-Disposition", disposition)

	// Cek apakah file lokal ada
	localPath := fileURL
	if strings.HasPrefix(localPath, "/") || strings.HasPrefix(localPath, "\\") {
		localPath = "." + localPath
	}
	if info, err := os.Stat(localPath); err == nil && !info.IsDir() {
		c.File(localPath)
		return
	}

	// Fallback jika file fisik tidak ada di disk (misal URL eksternal atau seed)
	content := []byte(fmt.Sprintf("=== DOKUMEN MATERI E-CLASS ===\n\nJudul: %s\nMata Pelajaran: %s\nDeskripsi: %s\nFile Asli: %s\nTanggal: %s\n",
		materi.Judul, materi.Pelajaran.Nama, materi.Deskripsi, materi.FileUrl, materi.CreatedAt.Format("02 January 2006")))
	c.Data(http.StatusOK, "application/octet-stream", content)
}
