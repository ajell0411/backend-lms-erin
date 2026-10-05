package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"e-class-backend/config"
	"e-class-backend/models"
)

type pelajaranInput struct {
	Nama          *string `json:"nama"`
	Kode          *string `json:"kode"`
	GuruIDs       []uint  `json:"guru_ids"`
	GuruIDsLegacy []uint  `json:"guruIds"`
}

func selectedGuruIDs(input pelajaranInput) ([]uint, bool) {
	if input.GuruIDs != nil {
		return input.GuruIDs, true
	}
	if input.GuruIDsLegacy != nil {
		return input.GuruIDsLegacy, true
	}
	return nil, false
}

func fetchGuru(ids []uint) ([]models.User, error) {
	if len(ids) == 0 {
		return []models.User{}, nil
	}
	unique := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		unique[id] = struct{}{}
	}
	var guru []models.User
	if err := config.DB.Where("id IN ? AND role = ?", ids, "guru").Order("nama").Find(&guru).Error; err != nil {
		return nil, err
	}
	if len(guru) != len(unique) {
		return nil, errInvalidGuruIDs
	}
	return guru, nil
}

var errInvalidGuruIDs = errors.New("Semua guru pengampu harus merupakan akun guru yang valid")

func pelajaranCodeExists(kode string, excludeID uint) (bool, error) {
	query := config.DB.Model(&models.Pelajaran{}).Where("lower(kode) = lower(?)", strings.TrimSpace(kode))
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

func pelajaranJSON(item models.Pelajaran) gin.H {
	guruIDs := make([]uint, 0, len(item.Guru))
	for _, guru := range item.Guru {
		guruIDs = append(guruIDs, guru.ID)
	}
	return gin.H{"id": item.ID, "nama": item.Nama, "kode": item.Kode, "guru": item.Guru, "guru_ids": guruIDs}
}

func CreatePelajaran(c *gin.Context) {
	var input pelajaranInput
	if err := c.ShouldBindJSON(&input); err != nil || input.Nama == nil || strings.TrimSpace(*input.Nama) == "" || input.Kode == nil || strings.TrimSpace(*input.Kode) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama dan kode pelajaran wajib diisi"})
		return
	}
	if exists, err := pelajaranCodeExists(*input.Kode, 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa kode pelajaran"})
		return
	} else if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Kode pelajaran sudah digunakan"})
		return
	}
	ids, _ := selectedGuruIDs(input)
	guruList, err := fetchGuru(ids)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidGuruIDs.Error()})
		return
	}
	item := models.Pelajaran{Nama: strings.TrimSpace(*input.Nama), Kode: strings.TrimSpace(*input.Kode)}
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		if len(guruList) > 0 {
			return tx.Model(&item).Association("Guru").Replace(guruList)
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan pelajaran"})
		return
	}
	config.DB.Preload("Guru").First(&item, item.ID)
	recordAktivitas(c, "menambahkan", "pelajaran", item.Nama)
	c.JSON(http.StatusCreated, pelajaranJSON(item))
}

func ListPelajaran(c *gin.Context) {
	var items []models.Pelajaran
	if err := config.DB.Preload("Guru").Order("nama").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat pelajaran"})
		return
	}
	data := make([]gin.H, 0, len(items))
	for _, item := range items {
		data = append(data, pelajaranJSON(item))
	}
	c.JSON(http.StatusOK, data)
}

func GetPelajaranByID(c *gin.Context) {
	var item models.Pelajaran
	if err := config.DB.Preload("Guru").First(&item, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pelajaran tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, pelajaranJSON(item))
}

func ListGuruPelajaran(c *gin.Context) {
	var item models.Pelajaran
	if err := config.DB.First(&item, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pelajaran tidak ditemukan"})
		return
	}
	var guru []models.User
	if err := config.DB.Model(&item).Association("Guru").Find(&guru); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat guru pengampu"})
		return
	}
	data := make([]gin.H, 0, len(guru))
	for _, item := range guru {
		data = append(data, gin.H{
			"id": item.ID, "nama": item.Nama, "email": item.Email, "nip": item.NIP,
			"status": item.Status, "foto_url": item.FotoURL,
		})
	}
	c.JSON(http.StatusOK, data)
}

func UpdatePelajaran(c *gin.Context) {
	var item models.Pelajaran
	if err := config.DB.First(&item, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pelajaran tidak ditemukan"})
		return
	}
	var input pelajaranInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data pelajaran tidak valid"})
		return
	}
	updates := map[string]interface{}{}
	if input.Nama != nil {
		if strings.TrimSpace(*input.Nama) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nama pelajaran wajib diisi"})
			return
		}
		updates["nama"] = strings.TrimSpace(*input.Nama)
	}
	if input.Kode != nil {
		if strings.TrimSpace(*input.Kode) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Kode pelajaran wajib diisi"})
			return
		}
		updates["kode"] = strings.TrimSpace(*input.Kode)
		if exists, err := pelajaranCodeExists(*input.Kode, item.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa kode pelajaran"})
			return
		} else if exists {
			c.JSON(http.StatusConflict, gin.H{"error": "Kode pelajaran sudah digunakan"})
			return
		}
	}
	ids, updateGuru := selectedGuruIDs(input)
	var guruList []models.User
	if updateGuru {
		var err error
		guruList, err = fetchGuru(ids)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidGuruIDs.Error()})
			return
		}
	}
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&item).Updates(updates).Error; err != nil {
				return err
			}
		}
		if updateGuru {
			return tx.Model(&item).Association("Guru").Replace(guruList)
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui pelajaran"})
		return
	}
	config.DB.Preload("Guru").First(&item, item.ID)
	recordAktivitas(c, "mengubah", "pelajaran", item.Nama)
	c.JSON(http.StatusOK, pelajaranJSON(item))
}

func DeletePelajaran(c *gin.Context) {
	var item models.Pelajaran
	if err := config.DB.First(&item, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pelajaran tidak ditemukan"})
		return
	}
	var materiCount, tugasCount, kuisCount int64
	if err := config.DB.Model(&models.Materi{}).Where("pelajaran_id = ?", item.ID).Count(&materiCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa penggunaan pelajaran"})
		return
	}
	if err := config.DB.Model(&models.Tugas{}).Where("pelajaran_id = ?", item.ID).Count(&tugasCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa penggunaan pelajaran"})
		return
	}
	if err := config.DB.Model(&models.Kuis{}).Where("pelajaran_id = ?", item.ID).Count(&kuisCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa penggunaan pelajaran"})
		return
	}
	if materiCount+tugasCount+kuisCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Pelajaran tidak bisa dihapus karena masih digunakan materi atau tugas"})
		return
	}
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&item).Association("Guru").Clear(); err != nil {
			return err
		}
		return tx.Delete(&item).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus pelajaran"})
		return
	}
	recordAktivitas(c, "menghapus", "pelajaran", item.Nama)
	c.JSON(http.StatusOK, gin.H{"message": "Pelajaran berhasil dihapus"})
}
