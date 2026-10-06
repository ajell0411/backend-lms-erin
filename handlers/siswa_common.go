package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
	"e-class-backend/models"
)

func sendSiswaSuccess(c *gin.Context, status int, message string, data interface{}) {
	c.JSON(status, gin.H{
		"success": true,
		"message": message,
		"data":    data,
	})
}

func sendSiswaError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"success": false,
		"message": message,
		"data":    nil,
	})
}

// getSiswaFromContext mengambil data siswa yang sedang login beserta kelas dan jurusannya
func getSiswaFromContext(c *gin.Context) (*models.User, bool) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		sendSiswaError(c, http.StatusUnauthorized, "Sesi Anda tidak valid atau telah berakhir")
		return nil, false
	}

	role := c.GetString("role")
	if role != "siswa" {
		sendSiswaError(c, http.StatusForbidden, "Akses hanya diperbolehkan untuk akun siswa")
		return nil, false
	}

	var user models.User
	if err := config.DB.Preload("Kelas.Jurusan").First(&user, userID).Error; err != nil {
		sendSiswaError(c, http.StatusNotFound, "Data siswa tidak ditemukan")
		return nil, false
	}

	if user.KelasID == nil || user.Kelas == nil {
		sendSiswaError(c, http.StatusForbidden, "Anda belum terdaftar dalam kelas mana pun")
		return nil, false
	}

	return &user, true
}
