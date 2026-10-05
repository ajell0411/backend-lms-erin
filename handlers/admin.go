package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"e-class-backend/config"
	"e-class-backend/models"
)

type adminInput struct {
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

func CreateAdmin(c *gin.Context) {
	var input adminInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data akun admin tidak valid"})
		return
	}

	if input.Nama == "" || input.Email == "" || input.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nama, email, dan password wajib diisi"})
		return
	}
	if input.Status != nil && !validateAccountStatus(*input.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status harus aktif atau nonaktif"})
		return
	}
	if !validNumericIdentifier(input.NIP) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NIP hanya boleh berisi angka"})
		return
	}
	status := "aktif"
	if input.Status != nil {
		status = *input.Status
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
		Status:   status,
	}
	user.Username = normalizedUsername(input.Username)
	if input.NIP != nil {
		user.NIP = *input.NIP
	}
	if input.Telepon != nil {
		user.Telepon = *input.Telepon
	}
	if input.FotoURL != nil {
		user.FotoURL = *input.FotoURL
	}

	if err := config.DB.Create(&user).Error; err != nil {
		respondAccountWriteError(c, err)
		return
	}
	recordAktivitas(c, "menambahkan", aktivitasJenisRole(user.Role), user.Nama)
	c.JSON(http.StatusCreated, akunJSON(user))
}

func ListAdmin(c *gin.Context) {
	var users []models.User
	if err := config.DB.Where("role = ?", "admin").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat daftar admin"})
		return
	}
	out := make([]gin.H, 0, len(users))
	for _, user := range users {
		out = append(out, akunJSON(user))
	}
	c.JSON(http.StatusOK, out)
}

func GetAdminByID(c *gin.Context) {
	var user models.User
	if err := config.DB.Where("role = ?", "admin").First(&user, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, akunJSON(user))
}

func UpdateAdmin(c *gin.Context) {
	var user models.User
	if err := config.DB.Where("role = ?", "admin").First(&user, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
		return
	}

	var input adminInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data akun admin tidak valid"})
		return
	}
	if input.Role != nil && !validAccountRole(*input.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role harus admin, admin_kurikulum, atau kepala_sekolah"})
		return
	}
	if input.Role != nil && *input.Role != user.Role && isCurrentUser(c, user.ID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin tidak dapat mengubah role akunnya sendiri"})
		return
	}
	if input.Status != nil && !validateAccountStatus(*input.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status harus aktif atau nonaktif"})
		return
	}
	if !validNumericIdentifier(input.NIP) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NIP hanya boleh berisi angka"})
		return
	}

	updates := map[string]interface{}{}
	if input.Nama != "" {
		updates["nama"] = input.Nama
	}
	if input.Email != "" {
		updates["email"] = input.Email
	}
	if input.Username != nil {
		updates["username"] = normalizedUsername(input.Username)
	}
	if input.Role != nil {
		updates["role"] = *input.Role
	}

	if input.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal hash password"})
			return
		}
		updates["password"] = string(hashed)
	}
	if input.NIP != nil {
		updates["NIP"] = *input.NIP
	}
	if input.Telepon != nil {
		updates["telepon"] = *input.Telepon
	}
	if input.Status != nil {
		updates["status"] = *input.Status
	}
	if input.FotoURL != nil {
		updates["foto_url"] = *input.FotoURL
	}

	if len(updates) > 0 {
		if err := config.DB.Model(&user).Updates(updates).Error; err != nil {
			respondAccountWriteError(c, err)
			return
		}
	}
	if err := config.DB.First(&user, user.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat akun admin yang diperbarui"})
		return
	}
	recordAktivitas(c, "mengubah", aktivitasJenisRole(user.Role), user.Nama)
	c.JSON(http.StatusOK, akunJSON(user))
}

func DeleteAdmin(c *gin.Context) {
	var user models.User
	if err := config.DB.Where("role = ?", "admin").First(&user, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin tidak ditemukan"})
		return
	}
	if err := config.DB.Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus admin"})
		return
	}
	recordAktivitas(c, "menghapus", aktivitasJenisRole(user.Role), user.Nama)
	c.JSON(http.StatusOK, gin.H{"message": "Berhasil dihapus"})
}
