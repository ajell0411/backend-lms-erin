package handlers

import (
	"net/http"
	"strings"

	"e-class-backend/config"
	"e-class-backend/models"
	"github.com/gin-gonic/gin"
)

type kelasInput struct {
	Nama              *string      `json:"nama"`
	Tingkat           *string      `json:"tingkat"`
	JurusanID         optionalUint `json:"jurusan_id"`
	JurusanIDLegacy   optionalUint `json:"jurusanId"`
	WaliKelasID       optionalUint `json:"wali_kelas_id"`
	WaliKelasIDLegacy optionalUint `json:"waliKelasId"`
}

func kelasJurusanID(input kelasInput) optionalUint {
	if input.JurusanID.Set {
		return input.JurusanID
	}
	return input.JurusanIDLegacy
}

func kelasWaliID(input kelasInput) optionalUint {
	if input.WaliKelasID.Set {
		return input.WaliKelasID
	}
	return input.WaliKelasIDLegacy
}

func validateWaliKelas(c *gin.Context, field optionalUint) bool {
	if !field.Set || field.Value == nil {
		return true
	}
	var count int64
	if err := config.DB.Model(&models.User{}).Where("id = ? AND role = ?", *field.Value, "guru").Count(&count).Error; err != nil || count == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Wali kelas harus merupakan akun guru yang valid"})
		return false
	}
	return true
}

func validateJurusan(c *gin.Context, id uint) bool {
	var count int64
	if err := config.DB.Model(&models.Jurusan{}).Where("id = ?", id).Count(&count).Error; err != nil || count == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jurusan tidak ditemukan"})
		return false
	}
	return true
}

func kelasNameExists(nama, tingkat string, jurusanID, excludeID uint) (bool, error) {
	query := config.DB.Model(&models.Kelas{}).Where("lower(nama) = lower(?) AND tingkat = ? AND jurusan_id = ?", strings.TrimSpace(nama), tingkat, jurusanID)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

func kelasJSON(item models.Kelas) gin.H {
	var jumlahSiswa int64
	config.DB.Model(&models.User{}).Where("role = ? AND kelas_id = ?", "siswa", item.ID).Count(&jumlahSiswa)
	waliNama := ""
	if item.WaliKelasID != nil {
		var wali models.User
		if err := config.DB.Select("nama").Where("id = ? AND role = ?", *item.WaliKelasID, "guru").First(&wali).Error; err == nil {
			waliNama = wali.Nama
		}
	}
	return gin.H{
		"id": item.ID, "nama": item.Nama, "tingkat": item.Tingkat,
		"jurusan_id": item.JurusanID, "jurusanId": item.JurusanID, "jurusan": item.Jurusan,
		"wali_kelas_id": item.WaliKelasID, "waliKelasId": item.WaliKelasID,
		"wali_kelas_nama": waliNama, "jumlah_siswa": jumlahSiswa,
	}
}

func loadKelas(item *models.Kelas) error {
	return config.DB.Preload("Jurusan").Preload("WaliKelas").First(item, item.ID).Error
}

func CreateKelas(c *gin.Context) {
	var input kelasInput
	if err := c.ShouldBindJSON(&input); err != nil || input.Nama == nil || strings.TrimSpace(*input.Nama) == "" || input.Tingkat == nil || strings.TrimSpace(*input.Tingkat) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama dan tingkat kelas wajib diisi"})
		return
	}
	if *input.Tingkat != "X" && *input.Tingkat != "XI" && *input.Tingkat != "XII" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tingkat kelas harus X, XI, atau XII"})
		return
	}
	jurusanID := kelasJurusanID(input)
	if !jurusanID.Set || jurusanID.Value == nil || *jurusanID.Value == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jurusan wajib dipilih"})
		return
	}
	if !validateJurusan(c, *jurusanID.Value) {
		return
	}
	if exists, err := kelasNameExists(*input.Nama, *input.Tingkat, *jurusanID.Value, 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa nama kelas"})
		return
	} else if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Nama kelas pada tingkat dan jurusan ini sudah digunakan"})
		return
	}
	if !validateWaliKelas(c, kelasWaliID(input)) {
		return
	}
	item := models.Kelas{Nama: strings.TrimSpace(*input.Nama), Tingkat: strings.TrimSpace(*input.Tingkat), JurusanID: *jurusanID.Value, WaliKelasID: kelasWaliID(input).Value}
	if err := config.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal menyimpan kelas"})
		return
	}
	_ = loadKelas(&item)
	recordAktivitas(c, "menambahkan", "kelas", item.Nama)
	c.JSON(http.StatusCreated, kelasJSON(item))
}

func ListKelas(c *gin.Context) {
	var items []models.Kelas
	query := config.DB.Preload("Jurusan").Preload("WaliKelas")
	if jurusanID := c.Query("jurusan_id"); jurusanID != "" {
		query = query.Where("jurusan_id = ?", jurusanID)
	}
	if err := query.Order("tingkat, nama").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat data kelas"})
		return
	}
	data := make([]gin.H, 0, len(items))
	for _, item := range items {
		data = append(data, kelasJSON(item))
	}
	c.JSON(http.StatusOK, data)
}

func GetKelasByID(c *gin.Context) {
	var item models.Kelas
	if err := config.DB.Preload("Jurusan").Preload("WaliKelas").First(&item, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kelas tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, kelasJSON(item))
}

func UpdateKelas(c *gin.Context) {
	var item models.Kelas
	if err := config.DB.First(&item, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kelas tidak ditemukan"})
		return
	}
	var input kelasInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data kelas tidak valid"})
		return
	}
	updates := map[string]interface{}{}
	if input.Nama != nil {
		if strings.TrimSpace(*input.Nama) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nama kelas wajib diisi"})
			return
		}
		updates["nama"] = strings.TrimSpace(*input.Nama)
	}
	if input.Tingkat != nil {
		if *input.Tingkat != "X" && *input.Tingkat != "XI" && *input.Tingkat != "XII" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tingkat kelas harus X, XI, atau XII"})
			return
		}
		updates["tingkat"] = *input.Tingkat
	}
	jurusanID := kelasJurusanID(input)
	if jurusanID.Set {
		if jurusanID.Value == nil || *jurusanID.Value == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Jurusan wajib dipilih"})
			return
		}
		if !validateJurusan(c, *jurusanID.Value) {
			return
		}
		updates["jurusan_id"] = *jurusanID.Value
	}
	waliID := kelasWaliID(input)
	if !validateWaliKelas(c, waliID) {
		return
	}
	if waliID.Set {
		updates["wali_kelas_id"] = waliID.Value
	}
	nama, tingkat, targetJurusan := item.Nama, item.Tingkat, item.JurusanID
	if value, ok := updates["nama"].(string); ok {
		nama = value
	}
	if value, ok := updates["tingkat"].(string); ok {
		tingkat = value
	}
	if value, ok := updates["jurusan_id"].(uint); ok {
		targetJurusan = value
	}
	if exists, err := kelasNameExists(nama, tingkat, targetJurusan, item.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa nama kelas"})
		return
	} else if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Nama kelas pada tingkat dan jurusan ini sudah digunakan"})
		return
	}
	if len(updates) > 0 {
		if err := config.DB.Model(&item).Updates(updates).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal memperbarui kelas"})
			return
		}
	}
	_ = loadKelas(&item)
	recordAktivitas(c, "mengubah", "kelas", item.Nama)
	c.JSON(http.StatusOK, kelasJSON(item))
}

func DeleteKelas(c *gin.Context) {
	var item models.Kelas
	if err := config.DB.First(&item, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kelas tidak ditemukan"})
		return
	}
	var count int64
	if err := config.DB.Model(&models.User{}).Where("role = ? AND kelas_id = ?", "siswa", item.ID).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa siswa di kelas"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Kelas tidak bisa dihapus karena masih memiliki siswa"})
		return
	}
	for _, dependency := range []struct {
		model interface{}
		label string
	}{
		{&models.Materi{}, "materi"}, {&models.Tugas{}, "tugas"}, {&models.Kuis{}, "kuis"},
	} {
		if err := config.DB.Model(dependency.model).Where("kelas_id = ?", item.ID).Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa penggunaan kelas"})
			return
		}
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Kelas tidak bisa dihapus karena masih digunakan " + dependency.label})
			return
		}
	}
	if err := config.DB.Delete(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus kelas"})
		return
	}
	recordAktivitas(c, "menghapus", "kelas", item.Nama)
	c.JSON(http.StatusOK, gin.H{"message": "Kelas berhasil dihapus"})
}
