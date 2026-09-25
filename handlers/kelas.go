package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
	"e-class-backend/models"
)

type kelasInput struct {
	Nama      string `json:"nama"`
	Tingkat   string `json:"tingkat"`
	JurusanID uint   `json:"jurusanId"`
}

func CreateKelas(c *gin.Context) {
	var input kelasInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	kelas := models.Kelas{
		Nama:      input.Nama,
		Tingkat:   input.Tingkat,
		JurusanID: input.JurusanID,
	}

	if err := config.DB.Create(&kelas).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	config.DB.Preload("Jurusan").First(&kelas, kelas.ID)
	c.JSON(http.StatusCreated, kelas)
}

func ListKelas(c *gin.Context) {
	var kelas []models.Kelas
	config.DB.Preload("Jurusan").Find(&kelas)
	c.JSON(http.StatusOK, kelas)
}

func GetKelasByID(c *gin.Context) {
	var kelas models.Kelas
	if err := config.DB.Preload("Jurusan").First(&kelas, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, kelas)
}

func UpdateKelas(c *gin.Context) {
	var kelas models.Kelas
	if err := config.DB.First(&kelas, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}

	var input kelasInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	kelas.Nama = input.Nama
	kelas.Tingkat = input.Tingkat
	kelas.JurusanID = input.JurusanID
	config.DB.Save(&kelas)
	config.DB.Preload("Jurusan").First(&kelas, kelas.ID)
	c.JSON(http.StatusOK, kelas)
}

func DeleteKelas(c *gin.Context) {
	if err := config.DB.Delete(&models.Kelas{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Berhasil dihapus"})
}