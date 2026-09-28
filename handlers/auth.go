package handlers

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"e-class-backend/config"
	"e-class-backend/models"
)

type loginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Login(c *gin.Context) {
	// Sama seperti middleware: tanpa JWT_SECRET, jangan lanjut
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Konfigurasi server bermasalah"})
		return
	}

	var input loginInput
	if err := c.ShouldBindJSON(&input); err != nil ||
		strings.TrimSpace(input.Email) == "" || input.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Email/username dan password wajib diisi"})
		return
	}

	var user models.User
	if err := config.DB.Where("email = ?", strings.TrimSpace(input.Email)).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Email atau password salah"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Email atau password salah"})
		return
	}

	// Nama klaim harus "userId" dan "role" supaya cocok dengan middleware.VerifyToken
	claims := jwt.MapClaims{
		"userId": user.ID,
		"role":   user.Role,
		"exp":    time.Now().Add(24 * time.Hour).Unix(),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Gagal membuat token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"token":   token,
		"data": gin.H{
			"id":    user.ID,
			"nama":  user.Nama,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}