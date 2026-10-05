package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"e-class-backend/config"
	"e-class-backend/models"
)

type siswaInput struct {
	Nama         *string `json:"nama"`
	Username     *string `json:"username"`
	Email        *string `json:"email"`
	Password     *string `json:"password"`
	NISN         *string `json:"nisn"`
	JenisKelamin *string `json:"jenis_kelamin"`
	TempatLahir  *string `json:"tempat_lahir"`
	TanggalLahir *string `json:"tanggal_lahir"`
	Alamat       *string `json:"alamat"`
	Telepon      *string `json:"telepon"`
	NamaWali     *string `json:"nama_wali"`
	TeleponWali  *string `json:"telepon_wali"`
	TahunMasuk   *int    `json:"tahun_masuk"`
	Status       *string `json:"status"`
	FotoURL      *string `json:"foto_url"`
	KelasID      *uint   `json:"kelas_id"`
}

func siswaJSON(user models.User) gin.H {
	username := ""
	if user.Username != nil {
		username = *user.Username
	}
	var kelasID *uint
	var namaKelas, namaJurusan string
	var jurusanID *uint
	if user.Kelas != nil {
		kelasID = &user.Kelas.ID
		namaKelas = user.Kelas.Nama
		jurusanID = &user.Kelas.JurusanID
		namaJurusan = user.Kelas.Jurusan.Nama
	}
	var nisn *string
	if user.NISN != nil && *user.NISN != "" {
		nisn = user.NISN
	}
	var tanggalLahir *string
	if user.TanggalLahir != nil {
		formatted := user.TanggalLahir.Format("2006-01-02")
		tanggalLahir = &formatted
	}
	return gin.H{
		"id": user.ID, "nama": user.Nama, "username": username, "email": user.Email, "role": user.Role,
		"nisn": nisn, "jenis_kelamin": user.JenisKelamin, "tempat_lahir": user.TempatLahir,
		"tanggal_lahir": tanggalLahir, "alamat": user.Alamat, "telepon": user.Telepon,
		"nama_wali": user.NamaWali, "telepon_wali": user.TeleponWali, "tahun_masuk": user.TahunMasuk,
		"status": user.Status, "foto_url": user.FotoURL, "kelas_id": kelasID,
		"nama_kelas": namaKelas, "jurusan_id": jurusanID, "nama_jurusan": namaJurusan,
		"created_at": user.CreatedAt,
	}
}

func validateSiswaInput(c *gin.Context, input siswaInput, creating bool) bool {
	if creating && (normalizeOptional(input.Nama) == "" || normalizeOptional(input.Username) == "" ||
		normalizeOptional(input.Email) == "" || input.Password == nil || *input.Password == "" ||
		normalizeOptional(input.Status) == "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama, username, email, password, dan status wajib diisi"})
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
	if input.TanggalLahir != nil && *input.TanggalLahir != "" {
		if _, err := time.Parse("2006-01-02", *input.TanggalLahir); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format tanggal lahir harus YYYY-MM-DD"})
			return false
		}
	}
	if input.KelasID != nil {
		var count int64
		if err := config.DB.Model(&models.Kelas{}).Where("id = ?", *input.KelasID).Count(&count).Error; err != nil || count == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Kelas tidak ditemukan"})
			return false
		}
	}
	return true
}

func parseTanggalLahir(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func respondSiswaWriteError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrDuplicatedKey) || isUniqueConstraintError(err) {
		c.JSON(http.StatusConflict, gin.H{"error": "Username, email, atau NISN sudah digunakan"})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "Data siswa tidak valid"})
}

func CreateSiswa(c *gin.Context) {
	var input siswaInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data siswa tidak valid"})
		return
	}
	if !validateSiswaInput(c, input, true) {
		return
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(*input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses password"})
		return
	}
	tanggalLahir, err := parseTanggalLahir(normalizeOptional(input.TanggalLahir))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format tanggal lahir tidak valid"})
		return
	}
	username := normalizeOptional(input.Username)
	user := models.User{
		Nama: normalizeOptional(input.Nama), Username: &username, Email: normalizeOptional(input.Email),
		Password: string(hashed), Role: "siswa", JenisKelamin: normalizeOptional(input.JenisKelamin),
		TempatLahir: normalizeOptional(input.TempatLahir), TanggalLahir: tanggalLahir,
		Alamat: normalizeOptional(input.Alamat), Telepon: normalizeOptional(input.Telepon),
		NamaWali: normalizeOptional(input.NamaWali), TeleponWali: normalizeOptional(input.TeleponWali),
		TahunMasuk: input.TahunMasuk, Status: normalizeOptional(input.Status), FotoURL: normalizeOptional(input.FotoURL),
		KelasID: input.KelasID,
	}
	if input.NISN != nil && strings.TrimSpace(*input.NISN) != "" {
		value := strings.TrimSpace(*input.NISN)
		user.NISN = &value
	}
	if err := config.DB.Create(&user).Error; err != nil {
		respondSiswaWriteError(c, err)
		return
	}
	if err := config.DB.Preload("Kelas.Jurusan").First(&user, user.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat siswa"})
		return
	}
	recordAktivitas(c, "menambahkan", "murid", user.Nama)
	c.JSON(http.StatusCreated, siswaJSON(user))
}

func ListSiswa(c *gin.Context) {
	query := config.DB.Model(&models.User{}).Where("users.role = ?", "siswa")
	if kelasID := c.Query("kelas_id"); kelasID != "" {
		query = query.Where("users.kelas_id = ?", kelasID)
	}
	if jurusanID := c.Query("jurusan_id"); jurusanID != "" {
		query = query.Joins("JOIN kelas ON kelas.id = users.kelas_id").Where("kelas.jurusan_id = ?", jurusanID)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("users.status = ?", status)
	}
	search := strings.TrimSpace(c.Query("search"))
	if search == "" {
		search = strings.TrimSpace(c.Query("q"))
	}
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("(users.nama LIKE ? OR users.nisn LIKE ?)", like, like)
	}
	var users []models.User
	if err := query.Preload("Kelas.Jurusan").Order("users.id").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat data siswa"})
		return
	}
	response := make([]gin.H, 0, len(users))
	for _, user := range users {
		response = append(response, siswaJSON(user))
	}
	c.JSON(http.StatusOK, response)
}

func GetSiswaByID(c *gin.Context) {
	var user models.User
	if err := config.DB.Preload("Kelas.Jurusan").Where("role = ?", "siswa").First(&user, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Siswa tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, siswaJSON(user))
}

func UpdateSiswa(c *gin.Context) {
	var user models.User
	if err := config.DB.Where("role = ?", "siswa").First(&user, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Siswa tidak ditemukan"})
		return
	}
	var input siswaInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data siswa tidak valid"})
		return
	}
	if !validateSiswaInput(c, input, false) {
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
	if input.NISN != nil {
		value := strings.TrimSpace(*input.NISN)
		if value == "" {
			updates["nisn"] = nil
		} else {
			updates["nisn"] = value
		}
	}
	if input.JenisKelamin != nil {
		updates["jenis_kelamin"] = *input.JenisKelamin
	}
	if input.TempatLahir != nil {
		updates["tempat_lahir"] = normalizeOptional(input.TempatLahir)
	}
	if input.TanggalLahir != nil {
		date, err := parseTanggalLahir(*input.TanggalLahir)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format tanggal lahir tidak valid"})
			return
		}
		updates["tanggal_lahir"] = date
	}
	if input.Alamat != nil {
		updates["alamat"] = normalizeOptional(input.Alamat)
	}
	if input.Telepon != nil {
		updates["telepon"] = normalizeOptional(input.Telepon)
	}
	if input.NamaWali != nil {
		updates["nama_wali"] = normalizeOptional(input.NamaWali)
	}
	if input.TeleponWali != nil {
		updates["telepon_wali"] = normalizeOptional(input.TeleponWali)
	}
	if input.TahunMasuk != nil {
		updates["tahun_masuk"] = *input.TahunMasuk
	}
	if input.Status != nil {
		updates["status"] = *input.Status
	}
	if input.FotoURL != nil {
		updates["foto_url"] = normalizeOptional(input.FotoURL)
	}
	if input.KelasID != nil {
		updates["kelas_id"] = *input.KelasID
	}
	if len(updates) > 0 {
		if err := config.DB.Model(&user).Updates(updates).Error; err != nil {
			respondSiswaWriteError(c, err)
			return
		}
	}
	if err := config.DB.Preload("Kelas.Jurusan").First(&user, user.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat siswa"})
		return
	}
	recordAktivitas(c, "mengubah", "murid", user.Nama)
	c.JSON(http.StatusOK, siswaJSON(user))
}

func DeleteSiswa(c *gin.Context) {
	var user models.User
	if err := config.DB.Where("role = ?", "siswa").First(&user, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Siswa tidak ditemukan"})
		return
	}
	result := config.DB.Where("role = ?", "siswa").Delete(&models.User{}, c.Param("id"))
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus siswa"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Siswa tidak ditemukan"})
		return
	}
	recordAktivitas(c, "menghapus", "murid", user.Nama)
	c.JSON(http.StatusOK, gin.H{"message": "Berhasil dihapus"})
}
