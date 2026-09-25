package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
	"e-class-backend/models"
)

type tugasInput struct {
	Judul       string `json:"judul"`
	Deskripsi   string `json:"deskripsi"`
	LampiranUrl string `json:"lampiranUrl"`
	Deadline    string `json:"deadline"`
	PelajaranID uint   `json:"pelajaranId"`
	KelasID     uint   `json:"kelasId"`
	GuruID      uint   `json:"guruId"`
}

// getKelasIDSiswa mengambil kelasId dari siswa yang sedang login
func getKelasIDSiswa(c *gin.Context) (uint, bool) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return 0, false
	}
	var u models.User
	if err := config.DB.First(&u, userID).Error; err != nil || u.KelasID == nil {
		return 0, false
	}
	return *u.KelasID, true
}

func CreateTugas(c *gin.Context) {
	var input tugasInput
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

	tugas := models.Tugas{
		Judul:       input.Judul,
		Deskripsi:   input.Deskripsi,
		LampiranUrl: input.LampiranUrl,
		Deadline:    deadline,
		PelajaranID: input.PelajaranID,
		KelasID:     input.KelasID,
		GuruID:      input.GuruID,
	}

	if err := config.DB.Create(&tugas).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, tugas)
}

func ListTugas(c *gin.Context) {
	var tugas []models.Tugas

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

	query.Find(&tugas)
	c.JSON(http.StatusOK, tugas)
}

func GetTugasByID(c *gin.Context) {
	var tugas models.Tugas
	if err := config.DB.First(&tugas, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}

	role := c.GetString("role")
	userID, _ := getUserIDFromContext(c)

	if role == "siswa" {
		kelasID, ok := getKelasIDSiswa(c)
		if !ok || kelasID != tugas.KelasID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tugas ini bukan untuk kelasmu"})
			return
		}
	}
	if role == "guru" && tugas.GuruID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bukan tugas milikmu"})
		return
	}

	c.JSON(http.StatusOK, tugas)
}

func UpdateTugas(c *gin.Context) {
	var tugas models.Tugas
	if err := config.DB.First(&tugas, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}

	role := c.GetString("role")
	userID, _ := getUserIDFromContext(c)
	if role == "guru" && tugas.GuruID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bukan tugas milikmu"})
		return
	}

	var input tugasInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deadline, err := time.Parse(time.RFC3339, input.Deadline)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format deadline salah, gunakan ISO 8601 (contoh: 2026-10-01T23:59:00Z)"})
		return
	}

	tugas.Judul = input.Judul
	tugas.Deskripsi = input.Deskripsi
	tugas.LampiranUrl = input.LampiranUrl
	tugas.Deadline = deadline
	tugas.PelajaranID = input.PelajaranID
	tugas.KelasID = input.KelasID
	// Hanya admin yang boleh memindahkan kepemilikan
	if role == "admin" && input.GuruID != 0 {
		tugas.GuruID = input.GuruID
	}

	config.DB.Save(&tugas)
	c.JSON(http.StatusOK, tugas)
}

func DeleteTugas(c *gin.Context) {
	var tugas models.Tugas
	if err := config.DB.First(&tugas, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}

	role := c.GetString("role")
	userID, _ := getUserIDFromContext(c)
	if role == "guru" && tugas.GuruID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bukan tugas milikmu"})
		return
	}

	var jumlahPengumpulan int64
	config.DB.Model(&models.PengumpulanTugas{}).
		Where("tugas_id = ?", tugas.ID).
		Count(&jumlahPengumpulan)
	if jumlahPengumpulan > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Tugas tidak bisa dihapus karena sudah ada pengumpulan siswa"})
		return
	}

	if err := config.DB.Delete(&tugas).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Berhasil dihapus"})
}