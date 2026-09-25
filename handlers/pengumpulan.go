package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
	"e-class-backend/models"
)

func getUserIDFromContext(c *gin.Context) (uint, bool) {
	raw, exists := c.Get("userId")
	if !exists {
		return 0, false
	}
	f, ok := raw.(float64)
	if !ok {
		return 0, false
	}
	return uint(f), true
}

type pengumpulanInput struct {
	TugasID        uint   `json:"tugasId"`
	FileJawabanUrl string `json:"fileJawabanUrl"`
}

type nilaiInput struct {
	Nilai    int    `json:"nilai"`
	Feedback string `json:"feedback"`
}

// cekGuruPemilikTugas memastikan tugas dari pengumpulan ini benar milik guru yang login.
// Dipakai berulang di GetPengumpulanByID, BeriNilaiPengumpulan, dan DeletePengumpulan.
func cekGuruPemilikTugas(c *gin.Context, tugasID uint, userID uint) (bool, *gin.Error) {
	var tugas models.Tugas
	if err := config.DB.First(&tugas, tugasID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tugas tidak ditemukan"})
		return false, nil
	}
	if tugas.GuruID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bukan tugas milikmu"})
		return false, nil
	}
	return true, nil
}

func CreatePengumpulan(c *gin.Context) {
	siswaID, ok := getUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User tidak dikenali"})
		return
	}

	var input pengumpulanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var tugas models.Tugas
	if err := config.DB.First(&tugas, input.TugasID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tugas tidak ditemukan"})
		return
	}

	var siswa models.User
	if err := config.DB.First(&siswa, siswaID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User tidak dikenali"})
		return
	}
	if siswa.KelasID == nil || *siswa.KelasID != tugas.KelasID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tugas ini bukan untuk kelasmu"})
		return
	}

	now := time.Now()
	terlambat := now.After(tugas.Deadline)

	var existing models.PengumpulanTugas
	err := config.DB.Where("tugas_id = ? AND siswa_id = ?", input.TugasID, siswaID).First(&existing).Error
	if err == nil {
		if existing.StatusPenilaian == "dinilai" {
			c.JSON(http.StatusConflict, gin.H{"error": "Tugas sudah dinilai, tidak bisa dikirim ulang"})
			return
		}
		existing.FileJawabanUrl = input.FileJawabanUrl
		existing.WaktuSubmit = now
		existing.Terlambat = terlambat
		config.DB.Save(&existing)
		c.JSON(http.StatusOK, existing)
		return
	}

	pengumpulan := models.PengumpulanTugas{
		TugasID:         input.TugasID,
		SiswaID:         siswaID,
		FileJawabanUrl:  input.FileJawabanUrl,
		StatusPenilaian: "belum_dinilai",
		WaktuSubmit:     now,
		Terlambat:       terlambat,
	}
	if err := config.DB.Create(&pengumpulan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, pengumpulan)
}

func ListPengumpulanByTugas(c *gin.Context) {
	tugasId := c.Query("tugasId")
	if tugasId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tugasId wajib diisi"})
		return
	}

	var tugas models.Tugas
	if err := config.DB.First(&tugas, tugasId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tugas tidak ditemukan"})
		return
	}

	role := c.GetString("role")
	userID, _ := getUserIDFromContext(c)
	if role == "guru" && tugas.GuruID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bukan tugas milikmu"})
		return
	}

	var list []models.PengumpulanTugas
	if err := config.DB.Where("tugas_id = ?", tugasId).Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func ListPengumpulanSaya(c *gin.Context) {
	siswaID, ok := getUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User tidak dikenali"})
		return
	}

	query := config.DB.Where("siswa_id = ?", siswaID)
	if tugasId := c.Query("tugasId"); tugasId != "" {
		query = query.Where("tugas_id = ?", tugasId)
	}

	var list []models.PengumpulanTugas
	if err := query.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// GetPengumpulanByID — sudah diperbaiki: guru sekarang juga dicek kepemilikan tugasnya
func GetPengumpulanByID(c *gin.Context) {
	var pengumpulan models.PengumpulanTugas
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
		ok, _ := cekGuruPemilikTugas(c, pengumpulan.TugasID, userID)
		if !ok {
			return // response error sudah dikirim di dalam cekGuruPemilikTugas
		}
	}

	c.JSON(http.StatusOK, pengumpulan)
}

func BeriNilaiPengumpulan(c *gin.Context) {
	var pengumpulan models.PengumpulanTugas
	if err := config.DB.First(&pengumpulan, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}

	userID, _ := getUserIDFromContext(c)
	ok, _ := cekGuruPemilikTugas(c, pengumpulan.TugasID, userID)
	if !ok {
		return
	}

	var input nilaiInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.Nilai < 0 || input.Nilai > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nilai harus 0 sampai 100"})
		return
	}

	pengumpulan.Nilai = &input.Nilai
	pengumpulan.Feedback = input.Feedback
	pengumpulan.StatusPenilaian = "dinilai"

	config.DB.Save(&pengumpulan)
	c.JSON(http.StatusOK, pengumpulan)
}

// DeletePengumpulan — sudah diperbaiki: guru sekarang wajib pemilik tugas untuk bisa hapus
func DeletePengumpulan(c *gin.Context) {
	var pengumpulan models.PengumpulanTugas
	if err := config.DB.First(&pengumpulan, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}

	role := c.GetString("role")
	userID, _ := getUserIDFromContext(c)

	switch role {
	case "siswa":
		if pengumpulan.SiswaID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak"})
			return
		}
		if pengumpulan.StatusPenilaian == "dinilai" {
			c.JSON(http.StatusConflict, gin.H{"error": "Tugas sudah dinilai, tidak bisa dihapus"})
			return
		}
	case "guru":
		ok, _ := cekGuruPemilikTugas(c, pengumpulan.TugasID, userID)
		if !ok {
			return
		}
	}

	if err := config.DB.Delete(&pengumpulan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Berhasil dihapus"})
}