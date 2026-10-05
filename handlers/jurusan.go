package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"e-class-backend/config"
	"e-class-backend/models"
)

type jurusanInput struct {
	Nama          *string `json:"nama"`
	Kode          *string `json:"kode"`
	KepalaJurusan *string `json:"kepala_jurusan"`
}

func jurusanJSON(item models.Jurusan) gin.H {
	var kelasCount, siswaCount int64
	config.DB.Model(&models.Kelas{}).Where("jurusan_id = ?", item.ID).Count(&kelasCount)
	config.DB.Model(&models.User{}).Where("role = ? AND kelas_id IN (?)", "siswa", config.DB.Model(&models.Kelas{}).Select("id").Where("jurusan_id = ?", item.ID)).Count(&siswaCount)
	return gin.H{
		"id": item.ID, "nama": item.Nama, "kode": item.Kode,
		"kepala_jurusan": item.KepalaJurusan, "jumlah_kelas": kelasCount,
		"jumlah_siswa": siswaCount,
	}
}

func CreateJurusan(c *gin.Context) {
	var input jurusanInput
	if err := c.ShouldBindJSON(&input); err != nil || input.Nama == nil || strings.TrimSpace(*input.Nama) == "" || input.Kode == nil || strings.TrimSpace(*input.Kode) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama dan kode jurusan wajib diisi"})
		return
	}
	item := models.Jurusan{Nama: strings.TrimSpace(*input.Nama), Kode: strings.TrimSpace(*input.Kode)}
	if input.KepalaJurusan != nil {
		item.KepalaJurusan = strings.TrimSpace(*input.KepalaJurusan)
	}
	if err := config.DB.Create(&item).Error; err != nil {
		if isUniqueConstraintError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Kode jurusan sudah digunakan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan jurusan"})
		return
	}
	recordAktivitas(c, "menambahkan", "jurusan", item.Nama)
	c.JSON(http.StatusCreated, jurusanJSON(item))
}

func ListJurusan(c *gin.Context) {
	var items []models.Jurusan
	if err := config.DB.Order("nama").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat jurusan"})
		return
	}
	data := make([]gin.H, 0, len(items))
	for _, item := range items {
		data = append(data, jurusanJSON(item))
	}
	c.JSON(http.StatusOK, data)
}

func GetJurusanByID(c *gin.Context) {
	var item models.Jurusan
	if err := config.DB.First(&item, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Jurusan tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, jurusanJSON(item))
}

func UpdateJurusan(c *gin.Context) {
	var item models.Jurusan
	if err := config.DB.First(&item, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Jurusan tidak ditemukan"})
		return
	}
	var input jurusanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data jurusan tidak valid"})
		return
	}
	updates := map[string]interface{}{}
	if input.Nama != nil {
		if strings.TrimSpace(*input.Nama) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nama jurusan wajib diisi"})
			return
		}
		updates["nama"] = strings.TrimSpace(*input.Nama)
	}
	if input.Kode != nil {
		if strings.TrimSpace(*input.Kode) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Kode jurusan wajib diisi"})
			return
		}
		updates["kode"] = strings.TrimSpace(*input.Kode)
	}
	if input.KepalaJurusan != nil {
		updates["kepala_jurusan"] = strings.TrimSpace(*input.KepalaJurusan)
	}
	if len(updates) > 0 {
		if err := config.DB.Model(&item).Updates(updates).Error; err != nil {
			if isUniqueConstraintError(err) {
				c.JSON(http.StatusConflict, gin.H{"error": "Kode jurusan sudah digunakan"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui jurusan"})
			return
		}
	}
	config.DB.First(&item, item.ID)
	recordAktivitas(c, "mengubah", "jurusan", item.Nama)
	c.JSON(http.StatusOK, jurusanJSON(item))
}

func DeleteJurusan(c *gin.Context) {
	var item models.Jurusan
	if err := config.DB.First(&item, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Jurusan tidak ditemukan"})
		return
	}
	var count int64
	if err := config.DB.Model(&models.Kelas{}).Where("jurusan_id = ?", item.ID).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa kelas jurusan"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Jurusan tidak bisa dihapus karena masih memiliki kelas"})
		return
	}
	if err := config.DB.Delete(&item).Error; err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus jurusan"})
		return
	}
	recordAktivitas(c, "menghapus", "jurusan", item.Nama)
	c.JSON(http.StatusOK, gin.H{"message": "Jurusan berhasil dihapus"})
}
