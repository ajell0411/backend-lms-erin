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
	PelajaranIDs []uint       `json:"pelajaran_ids"`
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
	pelajaranIDs := make([]uint, 0, len(user.Pelajaran))
	namaPelajaran := make([]string, 0, len(user.Pelajaran))
	if len(user.Pelajaran) > 0 {
		id := user.Pelajaran[0].ID
		pelajaranID = &id
		for _, pelajaran := range user.Pelajaran {
			pelajaranIDs = append(pelajaranIDs, pelajaran.ID)
			namaPelajaran = append(namaPelajaran, pelajaran.Nama)
		}
	}
	return gin.H{
		"id": user.ID, "nama": user.Nama, "username": username, "email": user.Email,
		"role": user.Role, "nip": user.NIP, "jenis_kelamin": user.JenisKelamin,
		"telepon": user.Telepon, "alamat": user.Alamat, "status": user.Status,
		"foto_url": user.FotoURL, "pelajaran_id": pelajaranID, "pelajaran_ids": pelajaranIDs,
		"nama_pelajaran": strings.Join(namaPelajaran, ", "), "created_at": user.CreatedAt,
	}
}

func selectedGuruLessons(input guruInput) ([]uint, bool) {
	if input.PelajaranIDs != nil {
		return input.PelajaranIDs, true
	}
	if !input.PelajaranID.Set {
		return nil, false
	}
	if input.PelajaranID.Value == nil {
		return []uint{}, true
	}
	return []uint{*input.PelajaranID.Value}, true
}

func validateGuruLessons(c *gin.Context, input guruInput) bool {
	ids, set := selectedGuruLessons(input)
	if !set {
		return true
	}
	if len(ids) > 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Guru hanya dapat memiliki maksimal 2 mata pelajaran"})
		return false
	}
	uniqueIDs := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Mata pelajaran tidak ditemukan"})
			return false
		}
		uniqueIDs[id] = struct{}{}
	}
	if len(uniqueIDs) != len(ids) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mata pelajaran tidak boleh dipilih lebih dari sekali"})
		return false
	}
	if len(ids) == 0 {
		return true
	}
	var count int64
	if err := config.DB.Model(&models.Pelajaran{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa mata pelajaran"})
		return false
	}
	if count != int64(len(ids)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mata pelajaran tidak ditemukan"})
		return false
	}
	return true
}

func validateGuruSubjectLimit(c *gin.Context, guruID uint, proposedIDs []uint, includeCurrentSubjects bool, excludedSubjectID uint, addNewSubject bool) bool {
	subjectIDs := make(map[uint]struct{})
	var assignedIDs []uint
	if err := config.DB.Model(&models.GuruKelas{}).Where("guru_id = ?", guruID).Distinct("pelajaran_id").Pluck("pelajaran_id", &assignedIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa penugasan mata pelajaran guru"})
		return false
	}
	for _, id := range assignedIDs {
		subjectIDs[id] = struct{}{}
	}
	if includeCurrentSubjects {
		query := config.DB.Table("pelajaran_guru").Where("user_id = ?", guruID)
		if excludedSubjectID > 0 {
			query = query.Where("pelajaran_id <> ?", excludedSubjectID)
		}
		var currentIDs []uint
		if err := query.Pluck("pelajaran_id", &currentIDs).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa mata pelajaran guru"})
			return false
		}
		for _, id := range currentIDs {
			subjectIDs[id] = struct{}{}
		}
	}
	for _, id := range proposedIDs {
		subjectIDs[id] = struct{}{}
	}
	subjectCount := len(subjectIDs)
	if addNewSubject {
		subjectCount++
	}
	if subjectCount > 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Guru hanya dapat mengajar maksimal 2 mata pelajaran"})
		return false
	}
	return true
}

func saveGuruLessons(db *gorm.DB, user *models.User, ids []uint, set bool) error {
	if !set {
		return nil
	}
	if len(ids) == 0 {
		return db.Model(user).Association("Pelajaran").Clear()
	}
	var pelajaran []models.Pelajaran
	if err := db.Where("id IN ?", ids).Find(&pelajaran).Error; err != nil {
		return err
	}
	return db.Model(user).Association("Pelajaran").Replace(&pelajaran)
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
	if !validNumericOptional(input.Telepon) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Telepon hanya boleh berisi angka"})
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
	println("DEBUG guru write error:", err.Error())
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
	if !validateGuruLessons(c, input) {
		return
	}
	if !validPassword(*input.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password minimal 8 karakter, hanya huruf dan angka"})
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
	lessonIDs, updateLessons := selectedGuruLessons(input)
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		return saveGuruLessons(tx, &user, lessonIDs, updateLessons)
	})
	if err != nil {
		respondGuruWriteError(c, err)
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
	if !validateGuruLessons(c, input) {
		return
	}
	lessonIDs, updateLessons := selectedGuruLessons(input)
	if updateLessons && !validateGuruSubjectLimit(c, user.ID, lessonIDs, false, 0, false) {
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
		if !validPassword(*input.Password) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password minimal 8 karakter, hanya huruf dan angka"})
			return
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(*input.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses password"})
			return
		}
		updates["password"] = string(hashed)
	}
	if !validNumericOptional(input.NIP) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NIP hanya boleh berisi angka"})
		return
	}
	if !validNumericOptional(input.Telepon) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Telepon hanya boleh berisi angka"})
		return
	}
	if input.NIP != nil {
		updates["NIP"] = normalizeOptional(input.NIP)
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
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&user).Updates(updates).Error; err != nil {
				return err
			}
		}
		return saveGuruLessons(tx, &user, lessonIDs, updateLessons)
	}); err != nil {
		respondGuruWriteError(c, err)
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
