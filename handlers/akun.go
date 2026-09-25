package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"e-class-backend/config"
	"e-class-backend/models"
)

type akunInput struct {
	Nama     string `json:"nama"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func akunJSON(u models.User) gin.H {
	return gin.H{
		"id":        u.ID,
		"nama":      u.Nama,
		"email":     u.Email,
		"role":      u.Role,
		"createdAt": u.CreatedAt,
	}
}

func CreateAkun(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in akunInput
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
			return
		}
		if in.Nama == "" || in.Email == "" || in.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nama, email, dan password wajib diisi"})
			return
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses password"})
			return
		}

		user := models.User{Nama: in.Nama, Email: in.Email, Password: string(hashed), Role: role}
		if err := config.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, akunJSON(user))
	}
}

func ListAkun(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var users []models.User
		if err := config.DB.Where("role = ?", role).Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out := make([]gin.H, 0, len(users))
		for _, u := range users {
			out = append(out, akunJSON(u))
		}
		c.JSON(http.StatusOK, out)
	}
}

func GetAkunByID(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.User
		if err := config.DB.Where("id = ? AND role = ?", c.Param("id"), role).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
			return
		}
		c.JSON(http.StatusOK, akunJSON(user))
	}
}

func UpdateAkun(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.User
		if err := config.DB.Where("id = ? AND role = ?", c.Param("id"), role).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
			return
		}

		var in akunInput
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
			return
		}

		// Hanya field yang dikirim yang diubah
		updates := map[string]interface{}{}
		if in.Nama != "" {
			updates["nama"] = in.Nama
		}
		if in.Email != "" {
			updates["email"] = in.Email
		}
		if in.Password != "" {
			hashed, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses password"})
				return
			}
			updates["password"] = string(hashed)
		}

		if len(updates) > 0 {
			if err := config.DB.Model(&user).Updates(updates).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
		config.DB.First(&user, user.ID)
		c.JSON(http.StatusOK, akunJSON(user))
	}
}

func DeleteAkun(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		res := config.DB.Where("id = ? AND role = ?", c.Param("id"), role).Delete(&models.User{})
		if res.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
			return
		}
		if res.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Berhasil dihapus"})
	}
}