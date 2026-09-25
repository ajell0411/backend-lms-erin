package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"e-class-backend/config"
	"e-class-backend/models"
)

type adminInput struct {
	Nama     string `json:"nama"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func CreateAdmin(c *gin.Context) {
	var input adminInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal hash password"})
		return
	}

	user := models.User{
		Nama:     input.Nama,
		Email:    input.Email,
		Password: string(hashed),
		Role:     "admin",
	}

	if err := config.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, user)
}

func ListAdmin(c *gin.Context) {
	var users []models.User
	config.DB.Where("role = ?", "admin").Find(&users)
	c.JSON(http.StatusOK, users)
}

func GetAdminByID(c *gin.Context) {
	var user models.User
	if err := config.DB.Where("role = ?", "admin").First(&user, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func UpdateAdmin(c *gin.Context) {
	var user models.User
	if err := config.DB.Where("role = ?", "admin").First(&user, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}

	var input adminInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user.Nama = input.Nama
	user.Email = input.Email

	if input.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal hash password"})
			return
		}
		user.Password = string(hashed)
	}

	config.DB.Save(&user)
	c.JSON(http.StatusOK, user)
}

func DeleteAdmin(c *gin.Context) {
	if err := config.DB.Where("role = ?", "admin").Delete(&models.User{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Berhasil dihapus"})
}