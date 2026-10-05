package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"e-class-backend/config"
	"e-class-backend/models"
)

type akunInput struct {
	Nama     string  `json:"nama"`
	Username *string `json:"username"`
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Role     *string `json:"role"`
	NIP      *string `json:"nip"`
	Telepon  *string `json:"telepon"`
	Status   *string `json:"status"`
	FotoURL  *string `json:"foto_url"`
}

func akunJSON(u models.User) gin.H {
	username := ""
	if u.Username != nil {
		username = *u.Username
	}
	return gin.H{
		"id":         u.ID,
		"nama":       u.Nama,
		"username":   username,
		"email":      u.Email,
		"role":       u.Role,
		"nip":        u.NIP,
		"telepon":    u.Telepon,
		"status":     u.Status,
		"foto_url":   u.FotoURL,
		"created_at": u.CreatedAt,
	}
}

func validAccountRole(role string) bool {
	return role == "admin" || role == "admin_kurikulum" || role == "kepala_sekolah"
}

func normalizedUsername(username *string) *string {
	if username == nil {
		return nil
	}
	value := strings.TrimSpace(*username)
	if value == "" {
		return nil
	}
	return &value
}

func respondAccountWriteError(c *gin.Context, err error) {
	message := strings.ToLower(err.Error())
	if (strings.Contains(message, "username") || strings.Contains(message, "email") || strings.Contains(message, "nisn")) &&
		(strings.Contains(message, "unique constraint") || strings.Contains(message, "duplicated key")) {
		c.JSON(http.StatusConflict, gin.H{"error": "Username, email, atau NISN sudah digunakan"})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal menyimpan akun"})
}

func isCurrentUser(c *gin.Context, id uint) bool {
	userID, exists := c.Get("userId")
	return exists && fmt.Sprint(userID) == fmt.Sprint(id)
}

func validateAccountStatus(status string) bool {
	return status == "aktif" || status == "nonaktif"
}

var numericIdentifier = regexp.MustCompile(`^[0-9]+$`)

func validNumericIdentifier(value *string) bool {
	return value == nil || strings.TrimSpace(*value) == "" || numericIdentifier.MatchString(strings.TrimSpace(*value))
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
		if in.Role != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Role akun ditentukan oleh endpoint"})
			return
		}
		if in.Status != nil && !validateAccountStatus(*in.Status) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status harus aktif atau nonaktif"})
			return
		}
		if !validNumericIdentifier(in.NIP) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "NIP hanya boleh berisi angka"})
			return
		}
		status := "aktif"
		if in.Status != nil {
			status = *in.Status
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses password"})
			return
		}

		user := models.User{
			Nama: in.Nama, Email: in.Email, Password: string(hashed), Role: role,
			Status: status,
		}
		user.Username = normalizedUsername(in.Username)
		if in.NIP != nil {
			user.NIP = *in.NIP
		}
		if in.Telepon != nil {
			user.Telepon = *in.Telepon
		}
		if in.FotoURL != nil {
			user.FotoURL = *in.FotoURL
		}
		if err := config.DB.Create(&user).Error; err != nil {
			respondAccountWriteError(c, err)
			return
		}
		recordAktivitas(c, "menambahkan", aktivitasJenisRole(role), user.Nama)
		c.JSON(http.StatusCreated, akunJSON(user))
	}
}

func ListAkun(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var users []models.User
		if err := config.DB.Where("role = ?", role).Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat daftar akun"})
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
		if in.Status != nil && !validateAccountStatus(*in.Status) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status harus aktif atau nonaktif"})
			return
		}
		if !validNumericIdentifier(in.NIP) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "NIP hanya boleh berisi angka"})
			return
		}
		if in.Role != nil && !validAccountRole(*in.Role) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Role harus admin, admin_kurikulum, atau kepala_sekolah"})
			return
		}
		if in.Role != nil && *in.Role != user.Role && isCurrentUser(c, user.ID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin tidak dapat mengubah role akunnya sendiri"})
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
		if in.Username != nil {
			updates["username"] = normalizedUsername(in.Username)
		}
		if in.Role != nil {
			updates["role"] = *in.Role
		}
		if in.Password != "" {
			hashed, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses password"})
				return
			}
			updates["password"] = string(hashed)
		}
		if in.NIP != nil {
			updates["nip"] = *in.NIP
		}
		if in.Telepon != nil {
			updates["telepon"] = *in.Telepon
		}
		if in.Status != nil {
			updates["status"] = *in.Status
		}
		if in.FotoURL != nil {
			updates["foto_url"] = *in.FotoURL
		}

		if len(updates) > 0 {
			if err := config.DB.Model(&user).Updates(updates).Error; err != nil {
				respondAccountWriteError(c, err)
				return
			}
		}
		if err := config.DB.First(&user, user.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat akun yang diperbarui"})
			return
		}
		recordAktivitas(c, "mengubah", aktivitasJenisRole(role), user.Nama)
		c.JSON(http.StatusOK, akunJSON(user))
	}
}

func DeleteAkun(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.User
		if err := config.DB.Where("id = ? AND role = ?", c.Param("id"), role).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Akun tidak ditemukan"})
			return
		}
		res := config.DB.Where("id = ? AND role = ?", c.Param("id"), role).Delete(&models.User{})
		if res.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus akun"})
			return
		}
		if res.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
			return
		}
		recordAktivitas(c, "menghapus", aktivitasJenisRole(role), user.Nama)
		c.JSON(http.StatusOK, gin.H{"message": "Berhasil dihapus"})
	}
}
