package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
	"e-class-backend/models"
)

type kuisInput struct {
	Judul       string `json:"judul"`
	Jenis       string `json:"jenis"`
	PelajaranID uint   `json:"pelajaranId"`
	KelasID     uint   `json:"kelasId"`
	Soal        string `json:"soal"`
	Deadline    string `json:"deadline"`
	GuruID      uint   `json:"guruId"`
}

func CreateKuis(c *gin.Context) {
	var input kuisInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deadline, err := time.Parse(time.RFC3339, input.Deadline)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format deadline salah, gunakan ISO 8601 (contoh: 2026-10-01T23:59:00Z)"})
		return
	}

	role := c.GetString("role")
	userID, _ := getUserIDFromContext(c)
	if role == "guru" {
		input.GuruID = userID
	} else if input.GuruID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "guruId wajib diisi"})
		return
	}

	if input.Jenis != "kuis" && input.Jenis != "ujian_harian" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "jenis harus 'kuis' atau 'ujian_harian'"})
		return
	}

	kuis := models.Kuis{
		Judul:       input.Judul,
		Jenis:       input.Jenis,
		PelajaranID: input.PelajaranID,
		KelasID:     input.KelasID,
		Soal:        input.Soal,
		Deadline:    deadline,
		GuruID:      input.GuruID,
	}

	if err := config.DB.Create(&kuis).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, kuis)
}

func ListKuis(c *gin.Context) {
	var kuis []models.Kuis

	role := c.GetString("role")
	userID, _ := getUserIDFromContext(c)

	query := config.DB
	switch role {
	case "siswa":
		kelasID, ok := getKelasIDSiswa(c)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Kamu belum terdaftar di kelas"})
			return
		}
		query = query.Where("kelas_id = ?", kelasID)
	case "guru":
		query = query.Where("guru_id = ?", userID)
	}

	if pelajaranId := c.Query("pelajaranId"); pelajaranId != "" {
		query = query.Where("pelajaran_id = ?", pelajaranId)
	}
	if role != "siswa" {
		if kelasId := c.Query("kelasId"); kelasId != "" {
			query = query.Where("kelas_id = ?", kelasId)
		}
	}

	query.Find(&kuis)
	c.JSON(http.StatusOK, kuis)
}

func GetKuisByID(c *gin.Context) {
	var kuis models.Kuis
	if err := config.DB.First(&kuis, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}

	role := c.GetString("role")
	userID, _ := getUserIDFromContext(c)

	if role == "siswa" {
		kelasID, ok := getKelasIDSiswa(c)
		if !ok || kelasID != kuis.KelasID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Kuis ini bukan untuk kelasmu"})
			return
		}
	}
	if role == "guru" && kuis.GuruID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bukan kuis milikmu"})
		return
	}

	c.JSON(http.StatusOK, kuis)
}

func UpdateKuis(c *gin.Context) {
	var kuis models.Kuis
	if err := config.DB.First(&kuis, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}

	role := c.GetString("role")
	userID, _ := getUserIDFromContext(c)
	if role == "guru" && kuis.GuruID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bukan kuis milikmu"})
		return
	}

	var input kuisInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deadline, err := time.Parse(time.RFC3339, input.Deadline)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format deadline salah, gunakan ISO 8601 (contoh: 2026-10-01T23:59:00Z)"})
		return
	}

	if input.Jenis != "kuis" && input.Jenis != "ujian_harian" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "jenis harus 'kuis' atau 'ujian_harian'"})
		return
	}

	kuis.Judul = input.Judul
	kuis.Jenis = input.Jenis
	kuis.PelajaranID = input.PelajaranID
	kuis.KelasID = input.KelasID
	kuis.Soal = input.Soal
	kuis.Deadline = deadline

	config.DB.Save(&kuis)
	c.JSON(http.StatusOK, kuis)
}

func DeleteKuis(c *gin.Context) {
	var kuis models.Kuis
	if err := config.DB.First(&kuis, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}

	role := c.GetString("role")
	userID, _ := getUserIDFromContext(c)
	if role == "guru" && kuis.GuruID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bukan kuis milikmu"})
		return
	}

	var jumlahPengumpulan int64
	config.DB.Model(&models.PengumpulanKuis{}).
		Where("kuis_id = ?", kuis.ID).
		Count(&jumlahPengumpulan)
	if jumlahPengumpulan > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Kuis tidak bisa dihapus karena sudah ada pengumpulan siswa"})
		return
	}

	if err := config.DB.Delete(&kuis).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Berhasil dihapus"})
}

// ===== Pengumpulan Kuis =====

type pengumpulanKuisInput struct {
	KuisID  uint   `json:"kuisId"`
	Jawaban string `json:"jawaban"`
}

type nilaiKuisInput struct {
	Nilai int `json:"nilai"`
}

func cekGuruPemilikKuis(c *gin.Context, kuisID uint, userID uint) bool {
	var kuis models.Kuis
	if err := config.DB.First(&kuis, kuisID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kuis tidak ditemukan"})
		return false
	}
	if kuis.GuruID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bukan kuis milikmu"})
		return false
	}
	return true
}

func CreatePengumpulanKuis(c *gin.Context) {
	siswaID, ok := getUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User tidak dikenali"})
		return
	}

	var input pengumpulanKuisInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var kuis models.Kuis
	if err := config.DB.First(&kuis, input.KuisID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kuis tidak ditemukan"})
		return
	}

	var siswa models.User
	if err := config.DB.First(&siswa, siswaID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User tidak dikenali"})
		return
	}
	if siswa.KelasID == nil || *siswa.KelasID != kuis.KelasID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Kuis ini bukan untuk kelasmu"})
		return
	}

	now := time.Now()
	if now.After(kuis.Deadline) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Waktu pengerjaan sudah berakhir"})
		return
	}

	var existing models.PengumpulanKuis
	err := config.DB.Where("kuis_id = ? AND siswa_id = ?", input.KuisID, siswaID).First(&existing).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Kamu sudah mengerjakan kuis ini"})
		return
	}

	pengumpulan := models.PengumpulanKuis{
		KuisID:       input.KuisID,
		SiswaID:      siswaID,
		Jawaban:      input.Jawaban,
		WaktuMulai:   now,
		WaktuSelesai: now,
	}
	if err := config.DB.Create(&pengumpulan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, pengumpulan)
}

func ListPengumpulanKuisByKuis(c *gin.Context) {
	kuisId := c.Query("kuisId")
	if kuisId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kuisId wajib diisi"})
		return
	}

	var kuis models.Kuis
	if err := config.DB.First(&kuis, kuisId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kuis tidak ditemukan"})
		return
	}

	role := c.GetString("role")
	userID, _ := getUserIDFromContext(c)
	if role == "guru" && kuis.GuruID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bukan kuis milikmu"})
		return
	}

	var list []models.PengumpulanKuis
	config.DB.Where("kuis_id = ?", kuisId).Find(&list)
	c.JSON(http.StatusOK, list)
}

func ListPengumpulanKuisSaya(c *gin.Context) {
	siswaID, ok := getUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User tidak dikenali"})
		return
	}

	query := config.DB.Where("siswa_id = ?", siswaID)
	if kuisId := c.Query("kuisId"); kuisId != "" {
		query = query.Where("kuis_id = ?", kuisId)
	}

	var list []models.PengumpulanKuis
	query.Find(&list)
	c.JSON(http.StatusOK, list)
}

func GetPengumpulanKuisByID(c *gin.Context) {
	var pengumpulan models.PengumpulanKuis
	if err := config.DB.First(&pengumpulan, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}

	role := c.GetString("role")
	userID, _ := getUserIDFromContext(c)

	if role == "siswa" && pengumpulan.SiswaID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak"})
		return
	}
	if role == "guru" {
		if !cekGuruPemilikKuis(c, pengumpulan.KuisID, userID) {
			return
		}
	}

	c.JSON(http.StatusOK, pengumpulan)
}

func BeriNilaiKuis(c *gin.Context) {
	var pengumpulan models.PengumpulanKuis
	if err := config.DB.First(&pengumpulan, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}

	userID, _ := getUserIDFromContext(c)
	if !cekGuruPemilikKuis(c, pengumpulan.KuisID, userID) {
		return
	}

	var input nilaiKuisInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.Nilai < 0 || input.Nilai > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nilai harus 0 sampai 100"})
		return
	}

	pengumpulan.Nilai = &input.Nilai
	config.DB.Save(&pengumpulan)
	c.JSON(http.StatusOK, pengumpulan)
}