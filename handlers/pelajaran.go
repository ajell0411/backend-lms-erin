package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
	"e-class-backend/models"
)

type pelajaranInput struct {
	Nama    string `json:"nama"`
	Kode    string `json:"kode"`
	GuruIDs []uint `json:"guruIds"`
}

func CreatePelajaran(c *gin.Context) {
	var input pelajaranInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var guruList []models.User
	if len(input.GuruIDs) > 0 {
		if err := config.DB.Where("id IN ? AND role = ?", input.GuruIDs, "guru").Find(&guruList).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	pelajaran := models.Pelajaran{
		Nama: input.Nama,
		Kode: input.Kode,
		Guru: guruList,
	}

	if err := config.DB.Create(&pelajaran).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, pelajaran)
}

func ListPelajaran(c *gin.Context) {
	var pelajaran []models.Pelajaran
	config.DB.Preload("Guru").Find(&pelajaran)
	c.JSON(http.StatusOK, pelajaran)
}

func GetPelajaranByID(c *gin.Context) {
	var pelajaran models.Pelajaran
	if err := config.DB.Preload("Guru").First(&pelajaran, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, pelajaran)
}

func UpdatePelajaran(c *gin.Context) {
	var pelajaran models.Pelajaran
	if err := config.DB.First(&pelajaran, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}

	var input pelajaranInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pelajaran.Nama = input.Nama
	pelajaran.Kode = input.Kode
	config.DB.Save(&pelajaran)

	if input.GuruIDs != nil {
		var guruList []models.User
		if len(input.GuruIDs) > 0 {
			config.DB.Where("id IN ? AND role = ?", input.GuruIDs, "guru").Find(&guruList)
		}
		config.DB.Model(&pelajaran).Association("Guru").Replace(guruList)
	}

	config.DB.Preload("Guru").First(&pelajaran, pelajaran.ID)
	c.JSON(http.StatusOK, pelajaran)
}

func DeletePelajaran(c *gin.Context) {
	if err := config.DB.Delete(&models.Pelajaran{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Berhasil dihapus"})
}