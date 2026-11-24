package handlers

import (
	"encoding/json"
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

type PengumpulanInfo struct {
	Nilai         *int     `json:"nilai,omitempty"`
	DikumpulkanAt string   `json:"dikumpulkan_at,omitempty"`
	NamaFile      string   `json:"nama_file,omitempty"`
	Catatan       string   `json:"catatan,omitempty"`
	Komentar      string   `json:"komentar,omitempty"`
	Files         []string `json:"files,omitempty"`
}

type LampiranInfo struct {
	Nama string `json:"nama,omitempty"`
	Url  string `json:"url,omitempty"`
}

type TugasSiswaItem struct {
	ID            uint               `json:"id"`
	Judul         string             `json:"judul"`
	Pelajaran     PelajaranSiswaItem `json:"pelajaran"`
	Tenggat       string             `json:"tenggat,omitempty"`
	CreatedAt     string             `json:"created_at,omitempty"`
	Status        string             `json:"status"` // "belum" | "dikumpulkan" | "dinilai" | "terlambat" | "proses" | "selesai"
	Progres       int                `json:"progres"`
	Nilai         *int               `json:"nilai,omitempty"`
	Pengumpulan   *PengumpulanInfo   `json:"pengumpulan,omitempty"`
	Deskripsi     string             `json:"deskripsi,omitempty"`
	Instruksi     string             `json:"instruksi,omitempty"`
	Kriteria      []string           `json:"kriteria,omitempty"`
	Guru          string             `json:"guru,omitempty"`
	Kelas         string             `json:"kelas,omitempty"`
	Bobot         int                `json:"bobot"`
	Lampiran      *LampiranInfo      `json:"lampiran,omitempty"`
	LampiranFiles []LampiranInfo     `json:"lampiran_files,omitempty"`
}

func buildTugasSiswaItem(t models.Tugas, siswaID uint) TugasSiswaItem {
	var pengumpulan models.PengumpulanTugas
	hasPengumpulan := config.DB.Where("tugas_id = ? AND siswa_id = ?", t.ID, siswaID).First(&pengumpulan).Error == nil

	var catatan models.SiswaPengumpulanCatatan
	if hasPengumpulan {
		config.DB.Where("tugas_id = ? AND siswa_id = ?", t.ID, siswaID).First(&catatan)
	}

	var meta models.TugasMetadata
	config.DB.Where("tugas_id = ?", t.ID).First(&meta)

	bobot := 10
	if meta.Bobot > 0 {
		bobot = meta.Bobot
	}
	instruksi := meta.Instruksi
	if instruksi == "" {
		instruksi = t.Deskripsi
	}
	var kriteria []string
	if meta.Kriteria != "" {
		_ = json.Unmarshal([]byte(meta.Kriteria), &kriteria)
	}
	if len(kriteria) == 0 {
		kriteria = []string{"Kesesuaian dengan instruksi tugas", "Kerapian penyusunan berkas", "Ketepatan waktu pengumpulan"}
	}

	status := "belum"
	progres := 0
	now := time.Now()
	var pInfo *PengumpulanInfo

	if hasPengumpulan {
		progres = 100
		if pengumpulan.Nilai != nil {
			status = "dinilai"
		} else {
			status = "dikumpulkan"
		}

		urls := guruDaftarFile(pengumpulan.FileJawabanUrl)
		namaFile := ""
		if len(urls) > 0 {
			namaFile = guruNamaTampil(urls[0])
		}
		names := make([]string, 0, len(urls))
		for _, fileURL := range urls {
			names = append(names, guruNamaTampil(fileURL))
		}
		pInfo = &PengumpulanInfo{
			Nilai:         pengumpulan.Nilai,
			DikumpulkanAt: pengumpulan.WaktuSubmit.Format(time.RFC3339),
			NamaFile:      namaFile,
			Catatan:       catatan.Catatan,
			Komentar:      pengumpulan.Feedback,
			Files:         names,
		}
	} else {
		if !t.Deadline.IsZero() && now.After(t.Deadline) {
			status = "terlambat"
		} else {
			status = "belum"
		}
	}

	var lampiran *LampiranInfo
	lampiranFiles := make([]LampiranInfo, 0)
	for _, fileURL := range guruDaftarFile(t.LampiranUrl) {
		item := LampiranInfo{Nama: guruNamaTampil(fileURL), Url: fileURL}
		lampiranFiles = append(lampiranFiles, item)
	}
	if len(lampiranFiles) > 0 {
		lampiran = &lampiranFiles[0]
	}

	tenggatStr := ""
	if !t.Deadline.IsZero() {
		tenggatStr = t.Deadline.Format(time.RFC3339)
	}

	return TugasSiswaItem{
		ID:    t.ID,
		Judul: t.Judul,
		Pelajaran: PelajaranSiswaItem{
			ID:   t.PelajaranID,
			Nama: t.Pelajaran.Nama,
			Kode: t.Pelajaran.Kode,
		},
		Tenggat:       tenggatStr,
		CreatedAt:     t.CreatedAt.Format(time.RFC3339),
		Status:        status,
		Progres:       progres,
		Nilai:         pengumpulan.Nilai,
		Pengumpulan:   pInfo,
		Deskripsi:     t.Deskripsi,
		Instruksi:     instruksi,
		Kriteria:      kriteria,
		Guru:          t.Guru.Nama,
		Kelas:         t.Kelas.Nama,
		Bobot:         bobot,
		Lampiran:      lampiran,
		LampiranFiles: lampiranFiles,
	}
}

func ListSiswaTugas(c *gin.Context) {
	siswa, ok := getSiswaFromContext(c)
	if !ok {
		return
	}

	query := config.DB.Preload("Pelajaran").Preload("Guru").Preload("Kelas").Where("kelas_id = ?", *siswa.KelasID)

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

	var tugasList []models.Tugas
	if err := query.Order("created_at DESC").Order("id DESC").Find(&tugasList).Error; err != nil {
		sendSiswaError(c, http.StatusInternalServerError, "Gagal mengambil daftar tugas")
		return
	}

	items := make([]TugasSiswaItem, 0, len(tugasList))
	for _, t := range tugasList {
		items = append(items, buildTugasSiswaItem(t, siswa.ID))
	}

	sendSiswaSuccess(c, http.StatusOK, "Berhasil memuat daftar tugas", items)
}

func GetSiswaTugasByID(c *gin.Context) {
	siswa, ok := getSiswaFromContext(c)
	if !ok {
		return
	}

	id := c.Param("id")
	var tugas models.Tugas
	if err := config.DB.Preload("Pelajaran").Preload("Guru").Preload("Kelas").First(&tugas, id).Error; err != nil {
		sendSiswaError(c, http.StatusNotFound, "Tugas tidak ditemukan")
		return
	}

	if tugas.KelasID != *siswa.KelasID {
		sendSiswaError(c, http.StatusForbidden, "Tugas ini bukan untuk kelas Anda")
		return
	}

	detail := buildTugasSiswaItem(tugas, siswa.ID)
	sendSiswaSuccess(c, http.StatusOK, "Berhasil memuat detail tugas", detail)
}

func UnduhSiswaTugasLampiran(c *gin.Context) {
	siswa, ok := getSiswaFromContext(c)
	if !ok {
		return
	}

	id := c.Param("id")
	var tugas models.Tugas
	if err := config.DB.Preload("Pelajaran").First(&tugas, id).Error; err != nil {
		sendSiswaError(c, http.StatusNotFound, "Tugas tidak ditemukan")
		return
	}

	if tugas.KelasID != *siswa.KelasID {
		sendSiswaError(c, http.StatusForbidden, "Tugas ini bukan untuk kelas Anda")
		return
	}

	files := guruDaftarFile(tugas.LampiranUrl)
	index := guruAtoiUint(c.Query("index"))
	if len(files) > 0 {
		if int(index) >= len(files) {
			sendSiswaError(c, http.StatusNotFound, "File lampiran tidak ditemukan")
			return
		}
		guruKirimFile(c, files[index])
		return
	}
	namaFile := filepath.Base(tugas.LampiranUrl)
	if namaFile == "" || namaFile == "." || namaFile == "/" || !strings.Contains(namaFile, ".") {
		namaFile = fmt.Sprintf("lampiran-tugas-%s.pdf", id)
	}

	disposition := fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", namaFile, url.PathEscape(namaFile))
	c.Header("Content-Disposition", disposition)

	localPath := tugas.LampiranUrl
	if strings.HasPrefix(localPath, "/") || strings.HasPrefix(localPath, "\\") {
		localPath = "." + localPath
	}
	if info, err := os.Stat(localPath); err == nil && !info.IsDir() {
		c.File(localPath)
		return
	}

	content := []byte(fmt.Sprintf("=== LAMPIRAN TUGAS E-CLASS ===\n\nJudul Tugas: %s\nPelajaran: %s\nDeskripsi: %s\n",
		tugas.Judul, tugas.Pelajaran.Nama, tugas.Deskripsi))
	c.Data(http.StatusOK, "application/octet-stream", content)
}

func KumpulkanSiswaTugas(c *gin.Context) {
	siswa, ok := getSiswaFromContext(c)
	if !ok {
		return
	}

	id := c.Param("id")
	var tugas models.Tugas
	if err := config.DB.Preload("Pelajaran").Preload("Guru").Preload("Kelas").First(&tugas, id).Error; err != nil {
		sendSiswaError(c, http.StatusNotFound, "Tugas tidak ditemukan")
		return
	}

	if tugas.KelasID != *siswa.KelasID {
		sendSiswaError(c, http.StatusForbidden, "Tugas ini bukan untuk kelas Anda")
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		sendSiswaError(c, http.StatusBadRequest, "File tugas wajib diunggah")
		return
	}
	fileHeaders := form.File["files"]
	if len(fileHeaders) == 0 {
		fileHeaders = form.File["file"]
	}
	if len(fileHeaders) == 0 {
		sendSiswaError(c, http.StatusBadRequest, "Pilih minimal satu file tugas")
		return
	}
	if len(fileHeaders) > 20 {
		sendSiswaError(c, http.StatusBadRequest, "Maksimal 20 file dalam satu kiriman")
		return
	}
	for _, fileHeader := range fileHeaders {
		if fileHeader.Size > 10*1024*1024 {
			sendSiswaError(c, http.StatusBadRequest, "Ukuran setiap file maksimal 10 MB")
			return
		}
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		if ext != ".zip" && ext != ".rar" && ext != ".pdf" && ext != ".doc" && ext != ".docx" && ext != ".ppt" && ext != ".pptx" && ext != ".xls" && ext != ".xlsx" && ext != ".txt" && ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
			sendSiswaError(c, http.StatusBadRequest, "Format file tidak didukung")
			return
		}
	}

	catatan := strings.TrimSpace(c.PostForm("catatan"))

	// Simpan file
	uploadDir := "./uploads/tugas"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		sendSiswaError(c, http.StatusInternalServerError, "Gagal menyiapkan direktori penyimpanan")
		return
	}

	fileUrls := make([]string, 0, len(fileHeaders))
	for _, fileHeader := range fileHeaders {
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		base := guruNamaFileAman.ReplaceAllString(strings.TrimSuffix(filepath.Base(fileHeader.Filename), ext), "_")
		if base == "" {
			base = "file"
		}
		uniqueFilename := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), base, ext)
		dst := filepath.Join(uploadDir, uniqueFilename)
		if err := c.SaveUploadedFile(fileHeader, dst); err != nil {
			for _, savedURL := range fileUrls {
				guruHapusFile(savedURL)
			}
			sendSiswaError(c, http.StatusInternalServerError, "Gagal menyimpan file tugas")
			return
		}
		fileUrls = append(fileUrls, fmt.Sprintf("/uploads/tugas/%s", uniqueFilename))
	}
	fileUrlBytes, _ := json.Marshal(fileUrls)
	fileUrl := string(fileUrlBytes)
	now := time.Now()
	isLate := !tugas.Deadline.IsZero() && now.After(tugas.Deadline)

	// Simpan atau update PengumpulanTugas
	var pengumpulan models.PengumpulanTugas
	err = config.DB.Where("tugas_id = ? AND siswa_id = ?", tugas.ID, siswa.ID).First(&pengumpulan).Error
	if err == nil {
		// Update pengumpulan yang sudah ada
		oldFileURLs := guruDaftarFile(pengumpulan.FileJawabanUrl)
		pengumpulan.FileJawabanUrl = fileUrl
		pengumpulan.WaktuSubmit = now
		pengumpulan.Terlambat = isLate
		if err := config.DB.Save(&pengumpulan).Error; err != nil {
			for _, savedURL := range fileUrls {
				guruHapusFile(savedURL)
			}
			sendSiswaError(c, http.StatusInternalServerError, "Gagal memperbarui pengumpulan tugas")
			return
		}
		for _, oldURL := range oldFileURLs {
			guruHapusFile(oldURL)
		}
	} else {
		// Buat baru
		pengumpulan = models.PengumpulanTugas{
			TugasID:         tugas.ID,
			SiswaID:         siswa.ID,
			FileJawabanUrl:  fileUrl,
			StatusPenilaian: "menunggu",
			WaktuSubmit:     now,
			Terlambat:       isLate,
		}
		if err := config.DB.Create(&pengumpulan).Error; err != nil {
			for _, savedURL := range fileUrls {
				guruHapusFile(savedURL)
			}
			sendSiswaError(c, http.StatusInternalServerError, "Gagal menyimpan pengumpulan tugas")
			return
		}
	}

	// Simpan catatan
	var catRecord models.SiswaPengumpulanCatatan
	if config.DB.Where("tugas_id = ? AND siswa_id = ?", tugas.ID, siswa.ID).First(&catRecord).Error == nil {
		catRecord.Catatan = catatan
		config.DB.Save(&catRecord)
	} else {
		catRecord = models.SiswaPengumpulanCatatan{
			TugasID: tugas.ID,
			SiswaID: siswa.ID,
			Catatan: catatan,
		}
		config.DB.Create(&catRecord)
	}

	detail := buildTugasSiswaItem(tugas, siswa.ID)
	sendSiswaSuccess(c, http.StatusOK, "Tugas berhasil dikumpulkan", detail)
}
