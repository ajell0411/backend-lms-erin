package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
	"e-class-backend/models"
)

type materiInput struct {
	Judul       string `json:"judul"`
	Deskripsi   string `json:"deskripsi"`
	FileUrl     string `json:"fileUrl"`
	PelajaranID uint   `json:"pelajaranId"`
	KelasID     uint   `json:"kelasId"`
	GuruID      uint   `json:"guruId"`
}

func CreateMateri(c *gin.Context) {
	var input materiInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	materi := models.Materi{
		Judul:       input.Judul,
		Deskripsi:   input.Deskripsi,
		FileUrl:     input.FileUrl,
		PelajaranID: input.PelajaranID,
		KelasID:     input.KelasID,
		GuruID:      input.GuruID,
	}

	if err := config.DB.Create(&materi).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, materi)
}

func ListMateri(c *gin.Context) {
	var materi []models.Materi

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

	query.Find(&materi)
	c.JSON(http.StatusOK, materi)
}

func GetMateriByID(c *gin.Context) {
	var materi models.Materi
	if err := config.DB.First(&materi, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}

	role := c.GetString("role")
	userID, _ := getUserIDFromContext(c)

	if role == "siswa" {
		kelasID, ok := getKelasIDSiswa(c)
		if !ok || kelasID != materi.KelasID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Materi ini bukan untuk kelasmu"})
			return
		}
	}
	if role == "guru" && materi.GuruID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bukan materi milikmu"})
		return
	}

	c.JSON(http.StatusOK, materi)
}

func UpdateMateri(c *gin.Context) {
	var materi models.Materi
	if err := config.DB.First(&materi, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}

	role := c.GetString("role")
	userID, _ := getUserIDFromContext(c)
	if role == "guru" && materi.GuruID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bukan materi milikmu"})
		return
	}

	var input materiInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	materi.Judul = input.Judul
	materi.Deskripsi = input.Deskripsi
	materi.FileUrl = input.FileUrl
	materi.PelajaranID = input.PelajaranID
	materi.KelasID = input.KelasID
	if role == "admin" && input.GuruID != 0 {
		materi.GuruID = input.GuruID
	}

	config.DB.Save(&materi)
	c.JSON(http.StatusOK, materi)
}

func DeleteMateri(c *gin.Context) {
	var materi models.Materi
	if err := config.DB.First(&materi, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}

	role := c.GetString("role")
	userID, _ := getUserIDFromContext(c)
	if role == "guru" && materi.GuruID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bukan materi milikmu"})
		return
	}

	if err := config.DB.Delete(&materi).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Berhasil dihapus"})
}