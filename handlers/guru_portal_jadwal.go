package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
	"e-class-backend/models"
)

type guruJadwalInput struct {
	KelasID     uint   `json:"kelas_id"`
	PelajaranID uint   `json:"pelajaran_id"`
	Hari        string `json:"hari"`
	JamMulai    string `json:"jam_mulai"`
	JamSelesai  string `json:"jam_selesai"`
	Ruangan     string `json:"ruangan"`
}

func guruPortalJadwalJSON(j models.Jadwal) gin.H {
	return gin.H{
		"id":             j.ID,
		"kelas_id":       j.KelasID,
		"kelas_nama":     j.Kelas.Nama,
		"pelajaran_id":   j.PelajaranID,
		"pelajaran_nama": j.Pelajaran.Nama,
		"hari":           j.Hari,
		"jam_mulai":      j.JamMulai,
		"jam_selesai":    j.JamSelesai,
		"ruangan":        j.Ruangan,
	}
}

func guruJadwalValid(in *guruJadwalInput) string {
	in.Hari = strings.ToLower(strings.TrimSpace(in.Hari))
	in.Ruangan = strings.TrimSpace(in.Ruangan)
	hariOK := false
	for _, h := range guruPortalHari {
		if h == in.Hari {
			hariOK = true
		}
	}
	if !hariOK {
		return "Hari tidak valid"
	}
	if in.KelasID == 0 || in.PelajaranID == 0 {
		return "Pilih kelas dan pelajaran"
	}
	mulai, err1 := time.Parse("15:04", in.JamMulai)
	selesai, err2 := time.Parse("15:04", in.JamSelesai)
	if err1 != nil || err2 != nil {
		return "Format jam harus JJ:MM"
	}
	if !selesai.After(mulai) {
		return "Jam selesai harus setelah jam mulai"
	}
	return ""
}

func ListGuruJadwal(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	q := config.DB.Preload("Kelas").Preload("Pelajaran").Where("guru_id = ?", guruID)
	if hari := strings.ToLower(c.Query("hari")); hari != "" {
		q = q.Where("hari = ?", hari)
	}
	var data []models.Jadwal
	q.Order("jam_mulai ASC").Find(&data)
	out := make([]gin.H, 0, len(data))
	for _, j := range data {
		out = append(out, guruPortalJadwalJSON(j))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}

func TambahGuruJadwal(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	var in guruJadwalInput
	if err := c.ShouldBindJSON(&in); err != nil {
		guruPortalGagal(c, http.StatusBadRequest, "Data tidak valid")
		return
	}
	if pesan := guruJadwalValid(&in); pesan != "" {
		guruPortalGagal(c, http.StatusBadRequest, pesan)
		return
	}
	j := models.Jadwal{
		GuruID: guruID, KelasID: in.KelasID, PelajaranID: in.PelajaranID,
		Hari: in.Hari, JamMulai: in.JamMulai, JamSelesai: in.JamSelesai, Ruangan: in.Ruangan,
	}
	if err := config.DB.Create(&j).Error; err != nil {
		guruPortalGagal(c, http.StatusInternalServerError, "Jadwal gagal disimpan")
		return
	}
	config.DB.Preload("Kelas").Preload("Pelajaran").First(&j, j.ID)
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": guruPortalJadwalJSON(j)})
}

func UbahGuruJadwal(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	pid, ok := guruPortalParamID(c)
	if !ok {
		return
	}
	var j models.Jadwal
	if err := config.DB.Where("id = ? AND guru_id = ?", pid, guruID).First(&j).Error; err != nil {
		guruPortalGagal(c, http.StatusNotFound, "Jadwal tidak ditemukan")
		return
	}
	var in guruJadwalInput
	if err := c.ShouldBindJSON(&in); err != nil {
		guruPortalGagal(c, http.StatusBadRequest, "Data tidak valid")
		return
	}
	if pesan := guruJadwalValid(&in); pesan != "" {
		guruPortalGagal(c, http.StatusBadRequest, pesan)
		return
	}
	config.DB.Model(&j).Updates(map[string]interface{}{
		"kelas_id": in.KelasID, "pelajaran_id": in.PelajaranID, "hari": in.Hari,
		"jam_mulai": in.JamMulai, "jam_selesai": in.JamSelesai, "ruangan": in.Ruangan,
	})
	config.DB.Preload("Kelas").Preload("Pelajaran").First(&j, j.ID)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": guruPortalJadwalJSON(j)})
}

func HapusGuruJadwal(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	pid, ok := guruPortalParamID(c)
	if !ok {
		return
	}
	res := config.DB.Where("id = ? AND guru_id = ?", pid, guruID).Delete(&models.Jadwal{})
	if res.RowsAffected == 0 {
		guruPortalGagal(c, http.StatusNotFound, "Jadwal tidak ditemukan")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Jadwal dihapus"})
}
