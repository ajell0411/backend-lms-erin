package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"e-class-backend/config"
	"e-class-backend/models"
)

type profileInput struct {
	Nama     string  `json:"nama"`
	Email    string  `json:"email"`
	Username *string `json:"username"`
	Telepon  string  `json:"telepon"`
	Alamat   string  `json:"alamat"`
	FotoURL  string  `json:"foto_url"`
}
type profilePasswordInput struct {
	PasswordLama string `json:"password_lama"`
	PasswordBaru string `json:"password_baru"`
}

func currentProfile(c *gin.Context) (models.User, bool) {
	id, ok := getUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return models.User{}, false
	}
	var user models.User
	if err := config.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Akun tidak ditemukan"})
		return models.User{}, false
	}
	return user, true
}
func profileJSON(user models.User) gin.H {
	username := ""
	if user.Username != nil {
		username = *user.Username
	}
	return gin.H{"id": user.ID, "nama": user.Nama, "username": username, "email": user.Email, "role": user.Role, "nip": user.NIP, "nisn": user.NISN, "telepon": user.Telepon, "alamat": user.Alamat, "foto_url": user.FotoURL, "status": user.Status, "created_at": user.CreatedAt}
}
func GetProfile(c *gin.Context) {
	user, ok := currentProfile(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, profileJSON(user))
}
func UpdateProfile(c *gin.Context) {
	user, ok := currentProfile(c)
	if !ok {
		return
	}
	var input profileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data profil tidak valid"})
		return
	}
	nama, email := strings.TrimSpace(input.Nama), strings.TrimSpace(input.Email)
	if nama == "" || email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama dan email wajib diisi"})
		return
	}
	updates := map[string]any{"nama": nama, "email": email, "telepon": strings.TrimSpace(input.Telepon), "alamat": strings.TrimSpace(input.Alamat), "foto_url": strings.TrimSpace(input.FotoURL)}
	if input.Username != nil {
		updates["username"] = normalizedUsername(input.Username)
	}
	if err := config.DB.Model(&user).Updates(updates).Error; err != nil {
		respondAccountWriteError(c, err)
		return
	}
	if err := config.DB.First(&user, user.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat profil"})
		return
	}
	c.JSON(http.StatusOK, profileJSON(user))
}
func UpdateProfilePassword(c *gin.Context) {
	user, ok := currentProfile(c)
	if !ok {
		return
	}
	var input profilePasswordInput
	if err := c.ShouldBindJSON(&input); err != nil || input.PasswordLama == "" || len(input.PasswordBaru) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Isi password lama dan password baru minimal 8 karakter"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.PasswordLama)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password lama tidak sesuai"})
		return
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(input.PasswordBaru), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses password"})
		return
	}
	if err := config.DB.Model(&user).Update("password", string(hashed)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Password berhasil diperbarui"})
}
