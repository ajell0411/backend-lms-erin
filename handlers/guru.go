package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"e-class-backend/config"
	"e-class-backend/models"
)

type guruInput struct {
	Nama         *string      `json:"nama"`
	Username     *string      `json:"username"`
	Email        *string      `json:"email"`
	Password     *string      `json:"password"`
	NIP          *string      `json:"nip"`
	JenisKelamin *string      `json:"jenis_kelamin"`
	Telepon      *string      `json:"telepon"`
	Alamat       *string      `json:"alamat"`
	Status       *string      `json:"status"`
	FotoURL      *string      `json:"foto_url"`
	PelajaranID  optionalUint `json:"pelajaran_id"`
}

type optionalUint struct {
	Set   bool
	Value *uint
}

func (value *optionalUint) UnmarshalJSON(data []byte) error {
	value.Set = true
	if string(data) == "null" {
		value.Value = nil
		return nil
	}
	var id uint
	if err := json.Unmarshal(data, &id); err != nil {
		return err
	}
	value.Value = &id
	return nil
}

func guruJSON(user models.User) gin.H {
	username := ""
	if user.Username != nil {
		username = *user.Username
	}
	var pelajaranID *uint
	namaPelajaran := ""
	if len(user.Pelajaran) > 0 {
		id := user.Pelajaran[0].ID
		pelajaranID = &id
		namaPelajaran = user.Pelajaran[0].Nama
	}
	return gin.H{
		"id": user.ID, "nama": user.Nama, "username": username, "email": user.Email,
		"role": user.Role, "nip": user.NIP, "jenis_kelamin": user.JenisKelamin,
		"telepon": user.Telepon, "alamat": user.Alamat, "status": user.Status,
		"foto_url": user.FotoURL, "pelajaran_id": pelajaranID,
		"nama_pelajaran": namaPelajaran, "created_at": user.CreatedAt,
	}
}

func validateGuruLesson(c *gin.Context, field optionalUint) bool {
	if !field.Set || field.Value == nil {
		return true
	}
	var count int64
	if err := config.DB.Model(&models.Pelajaran{}).Where("id = ?", *field.Value).Count(&count).Error; err != nil || count == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mata pelajaran tidak ditemukan"})
		return false
	}
	return true
}

func saveGuruLesson(user *models.User, field optionalUint) error {
	if !field.Set {
		return nil
	}
	if field.Value == nil {
		return config.DB.Model(user).Association("Pelajaran").Clear()
	}
	var pelajaran models.Pelajaran
	if err := config.DB.First(&pelajaran, *field.Value).Error; err != nil {
		return err
	}
	return config.DB.Model(user).Association("Pelajaran").Replace(&pelajaran)
}

func validJenisKelamin(value string) bool {
	return value == "L" || value == "P"
}

func normalizeOptional(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func validateGuruInput(c *gin.Context, input guruInput, creating bool) bool {
	if creating && (normalizeOptional(input.Nama) == "" || normalizeOptional(input.Username) == "" ||
		normalizeOptional(input.Email) == "" || input.Password == nil || *input.Password == "" ||
		normalizeOptional(input.Status) == "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama, username, email, password, dan status wajib diisi"})
		return false
	}
	if !validNumericIdentifier(input.NIP) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NIP hanya boleh berisi angka"})
		return false
	}
	if input.Status != nil && *input.Status != "aktif" && *input.Status != "nonaktif" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status harus aktif atau nonaktif"})
		return false
	}
	if input.JenisKelamin != nil && *input.JenisKelamin != "" && !validJenisKelamin(*input.JenisKelamin) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jenis kelamin harus L atau P"})
		return false
	}
	return true
}

func respondGuruWriteError(c *gin.Context, err error) {
	if isUniqueConstraintError(err) {
		c.JSON(http.StatusConflict, gin.H{"error": "Username atau email sudah digunakan"})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "Data guru tidak valid"})
}

func CreateGuru(c *gin.Context) {
	var input guruInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data guru tidak valid"})
		return
	}
	if !validateGuruInput(c, input, true) {
		return
	}
	if !validateGuruLesson(c, input.PelajaranID) {
		return
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(*input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses password"})
		return
	}
	username := normalizeOptional(input.Username)
	user := models.User{
		Nama: normalizeOptional(input.Nama), Username: &username, Email: normalizeOptional(input.Email),
		Password: string(hashed), Role: "guru", NIP: normalizeOptional(input.NIP),
		JenisKelamin: normalizeOptional(input.JenisKelamin), Telepon: normalizeOptional(input.Telepon),
		Alamat: normalizeOptional(input.Alamat), Status: normalizeOptional(input.Status),
		FotoURL: normalizeOptional(input.FotoURL),
	}
	if err := config.DB.Create(&user).Error; err != nil {
		respondGuruWriteError(c, err)
		return
	}
	if err := saveGuruLesson(&user, input.PelajaranID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal menyimpan mata pelajaran guru"})
		return
	}
	if err := config.DB.Preload("Pelajaran").First(&user, user.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat data guru"})
		return
	}
	recordAktivitas(c, "menambahkan", "guru", user.Nama)
	c.JSON(http.StatusCreated, guruJSON(user))
}

func ListGuru(c *gin.Context) {
	query := config.DB.Model(&models.User{}).Where("role = ?", "guru")
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + search + "%"
		query = query.Where("(nama LIKE ? OR email LIKE ?)", like, like)
	}
	if pelajaranID := c.Query("pelajaran_id"); pelajaranID != "" {
		query = query.Joins("JOIN pelajaran_guru ON pelajaran_guru.user_id = users.id").Where("pelajaran_guru.pelajaran_id = ?", pelajaranID).Distinct("users.*")
	}
	var users []models.User
	if err := query.Preload("Pelajaran").Order("id").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat data guru"})
		return
	}
	response := make([]gin.H, 0, len(users))
	for _, user := range users {
		response = append(response, guruJSON(user))
	}
	c.JSON(http.StatusOK, response)
}

func GetGuruByID(c *gin.Context) {
	var user models.User
	if err := config.DB.Preload("Pelajaran").Where("role = ?", "guru").First(&user, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Guru tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, guruJSON(user))
}

func UpdateGuru(c *gin.Context) {
	var user models.User
	if err := config.DB.Where("role = ?", "guru").First(&user, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Guru tidak ditemukan"})
		return
	}
	var input guruInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data guru tidak valid"})
		return
	}
	if !validateGuruInput(c, input, false) {
		return
	}
	if !validateGuruLesson(c, input.PelajaranID) {
		return
	}
	updates := map[string]interface{}{}
	if input.Nama != nil {
		updates["nama"] = normalizeOptional(input.Nama)
	}
	if input.Username != nil {
		value := normalizeOptional(input.Username)
		if value == "" {
			updates["username"] = nil
		} else {
			updates["username"] = value
		}
	}
	if input.Email != nil {
		updates["email"] = normalizeOptional(input.Email)
	}
	if input.Password != nil && *input.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(*input.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses password"})
			return
		}
		updates["password"] = string(hashed)
	}
	if input.NIP != nil {
		updates["nip"] = normalizeOptional(input.NIP)
	}
	if input.JenisKelamin != nil {
		updates["jenis_kelamin"] = *input.JenisKelamin
	}
	if input.Telepon != nil {
		updates["telepon"] = normalizeOptional(input.Telepon)
	}
	if input.Alamat != nil {
		updates["alamat"] = normalizeOptional(input.Alamat)
	}
	if input.Status != nil {
		updates["status"] = *input.Status
	}
	if input.FotoURL != nil {
		updates["foto_url"] = normalizeOptional(input.FotoURL)
	}
	if len(updates) > 0 {
		if err := config.DB.Model(&user).Updates(updates).Error; err != nil {
			respondGuruWriteError(c, err)
			return
		}
	}
	if err := saveGuruLesson(&user, input.PelajaranID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal menyimpan mata pelajaran guru"})
		return
	}
	if err := config.DB.Preload("Pelajaran").First(&user, user.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat guru"})
		return
	}
	recordAktivitas(c, "mengubah", "guru", user.Nama)
	c.JSON(http.StatusOK, guruJSON(user))
}

func DeleteGuru(c *gin.Context) {
	var user models.User
	if err := config.DB.Where("role = ?", "guru").First(&user, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Guru tidak ditemukan"})
		return
	}
	result := config.DB.Where("role = ?", "guru").Delete(&models.User{}, c.Param("id"))
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus guru"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Guru tidak ditemukan"})
		return
	}
	recordAktivitas(c, "menghapus", "guru", user.Nama)
	c.JSON(http.StatusOK, gin.H{"message": "Berhasil dihapus"})
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint") || strings.Contains(message, "duplicated key") ||
		strings.Contains(message, "unique failed") || errorsIsDuplicated(err)
}

func errorsIsDuplicated(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey)
}
