package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
	"e-class-backend/models"
)

type jurusanInput struct {
	Nama string `json:"nama"`
	Kode string `json:"kode"`
}

func CreateJurusan(c *gin.Context) {
	var input jurusanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	jurusan := models.Jurusan{
		Nama: input.Nama,
		Kode: input.Kode,
	}

	if err := config.DB.Create(&jurusan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, jurusan)
}

func ListJurusan(c *gin.Context) {
	var jurusan []models.Jurusan
	config.DB.Find(&jurusan)
	c.JSON(http.StatusOK, jurusan)
}

func GetJurusanByID(c *gin.Context) {
	var jurusan models.Jurusan
	if err := config.DB.First(&jurusan, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, jurusan)
}

func UpdateJurusan(c *gin.Context) {
	var jurusan models.Jurusan
	if err := config.DB.First(&jurusan, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}

	var input jurusanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	jurusan.Nama = input.Nama
	jurusan.Kode = input.Kode
	config.DB.Save(&jurusan)
	c.JSON(http.StatusOK, jurusan)
}

func DeleteJurusan(c *gin.Context) {
	if err := config.DB.Delete(&models.Jurusan{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Berhasil dihapus"})
}