package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
	"e-class-backend/models"
)

type guruKelasInput struct {
	KelasID     uint `json:"kelas_id"`
	PelajaranID uint `json:"pelajaran_id"`
}

func guruKelasJSON(gk models.GuruKelas) gin.H {
	var jumlah int64
	config.DB.Model(&models.User{}).Where("role = ? AND kelas_id = ?", "siswa", gk.KelasID).Count(&jumlah)
	var siswa []models.User
	config.DB.Where("role = ? AND kelas_id = ?", "siswa", gk.KelasID).Order("nama ASC").Limit(3).Find(&siswa)
	avatar := []gin.H{}
	for _, s := range siswa {
		avatar = append(avatar, gin.H{"nama": s.Nama, "foto_url": s.FotoURL})
	}
	return gin.H{
		"id":             gk.ID,
		"kelas_id":       gk.KelasID,
		"kelas_nama":     gk.Kelas.Nama,
		"tingkat":        gk.Kelas.Tingkat,
		"pelajaran_id":   gk.PelajaranID,
		"pelajaran_nama": gk.Pelajaran.Nama,
		"jumlah_siswa":   jumlah,
		"siswa_preview":  avatar,
	}
}

func ListGuruKelas(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	var data []models.GuruKelas
	config.DB.Preload("Kelas").Preload("Pelajaran").Where("guru_id = ?", guruID).Order("id ASC").Find(&data)
	out := make([]gin.H, 0, len(data))
	for _, gk := range data {
		out = append(out, guruKelasJSON(gk))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}

func DetailGuruKelas(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	pid, ok := guruPortalParamID(c)
	if !ok {
		return
	}
	var gk models.GuruKelas
	if err := config.DB.Preload("Kelas").Preload("Pelajaran").
		Where("id = ? AND guru_id = ?", pid, guruID).First(&gk).Error; err != nil {
		guruPortalGagal(c, http.StatusNotFound, "Kelas tidak ditemukan")
		return
	}

	var siswa []models.User
	config.DB.Where("role = ? AND kelas_id = ?", "siswa", gk.KelasID).Order("nama ASC").Find(&siswa)

	var tugas []models.Tugas
	config.DB.Where("guru_id = ? AND kelas_id = ? AND pelajaran_id = ?", guruID, gk.KelasID, gk.PelajaranID).Find(&tugas)
	tugasIDs := []uint{}
	tugasJudul := map[uint]string{}
	for _, t := range tugas {
		tugasIDs = append(tugasIDs, t.ID)
		tugasJudul[t.ID] = t.Judul
	}
	var kuis []models.Kuis
	config.DB.Where("guru_id = ? AND kelas_id = ? AND pelajaran_id = ?", guruID, gk.KelasID, gk.PelajaranID).Find(&kuis)
	kuisIDs := []uint{}
	kuisJudul := map[uint]string{}
	for _, k := range kuis {
		kuisIDs = append(kuisIDs, k.ID)
		kuisJudul[k.ID] = k.Judul
	}

	rows := make([]gin.H, 0, len(siswa))
	for _, s := range siswa {
		row := gin.H{
			"id": s.ID, "nama": s.Nama, "nisn": guruPortalString(s.NISN), "foto_url": s.FotoURL,
			"tugas_judul": "", "tugas_nilai": nil, "asesmen_judul": "", "asesmen_nilai": nil,
		}
		if len(tugasIDs) > 0 {
			var pt models.PengumpulanTugas
			if err := config.DB.Where("siswa_id = ? AND tugas_id IN ?", s.ID, tugasIDs).
				Order("waktu_submit DESC").Limit(1).Find(&pt).Error; err == nil && pt.ID != 0 {
				row["tugas_judul"] = tugasJudul[pt.TugasID]
				row["tugas_nilai"] = pt.Nilai
			}
		}
		if len(kuisIDs) > 0 {
			var sesi models.SiswaAsesmenSesi
			if err := config.DB.Where("siswa_id = ? AND kuis_id IN ? AND sudah_dikirim = ?", s.ID, kuisIDs, true).
				Order("dikirim_at DESC").Limit(1).Find(&sesi).Error; err == nil && sesi.ID != 0 {
				row["asesmen_judul"] = kuisJudul[sesi.KuisID]
				row["asesmen_nilai"] = sesi.Nilai
			}
		}
		rows = append(rows, row)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"id":             gk.ID,
		"kelas_id":       gk.KelasID,
		"kelas_nama":     gk.Kelas.Nama,
		"tingkat":        gk.Kelas.Tingkat,
		"pelajaran_id":   gk.PelajaranID,
		"pelajaran_nama": gk.Pelajaran.Nama,
		"jumlah_siswa":   len(siswa),
		"siswa":          rows,
	}})
}

func TambahGuruKelas(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	var in guruKelasInput
	if err := c.ShouldBindJSON(&in); err != nil || in.KelasID == 0 || in.PelajaranID == 0 {
		guruPortalGagal(c, http.StatusBadRequest, "Pilih kelas dan pelajaran")
		return
	}
	var kelas models.Kelas
	if err := config.DB.First(&kelas, in.KelasID).Error; err != nil {
		guruPortalGagal(c, http.StatusNotFound, "Kelas tidak ditemukan")
		return
	}
	var pel models.Pelajaran
	if err := config.DB.First(&pel, in.PelajaranID).Error; err != nil {
		guruPortalGagal(c, http.StatusNotFound, "Pelajaran tidak ditemukan")
		return
	}
	gk := models.GuruKelas{GuruID: guruID, KelasID: in.KelasID, PelajaranID: in.PelajaranID}
	if err := config.DB.Create(&gk).Error; err != nil {
		guruPortalGagal(c, http.StatusConflict, "Kelas dan pelajaran ini sudah ada di daftarmu")
		return
	}
	config.DB.Preload("Kelas").Preload("Pelajaran").First(&gk, gk.ID)
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": guruKelasJSON(gk)})
}

func UbahGuruKelas(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	pid, ok := guruPortalParamID(c)
	if !ok {
		return
	}
	var gk models.GuruKelas
	if err := config.DB.Where("id = ? AND guru_id = ?", pid, guruID).First(&gk).Error; err != nil {
		guruPortalGagal(c, http.StatusNotFound, "Kelas tidak ditemukan")
		return
	}
	var in guruKelasInput
	if err := c.ShouldBindJSON(&in); err != nil {
		guruPortalGagal(c, http.StatusBadRequest, "Data tidak valid")
		return
	}
	update := map[string]interface{}{}
	if in.KelasID > 0 {
		update["kelas_id"] = in.KelasID
	}
	if in.PelajaranID > 0 {
		update["pelajaran_id"] = in.PelajaranID
	}
	if len(update) == 0 {
		guruPortalGagal(c, http.StatusBadRequest, "Tidak ada perubahan")
		return
	}
	if err := config.DB.Model(&gk).Updates(update).Error; err != nil {
		guruPortalGagal(c, http.StatusConflict, "Kelas dan pelajaran ini sudah ada di daftarmu")
		return
	}
	config.DB.Preload("Kelas").Preload("Pelajaran").First(&gk, gk.ID)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": guruKelasJSON(gk)})
}

func HapusGuruKelas(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	pid, ok := guruPortalParamID(c)
	if !ok {
		return
	}
	res := config.DB.Where("id = ? AND guru_id = ?", pid, guruID).Delete(&models.GuruKelas{})
	if res.RowsAffected == 0 {
		guruPortalGagal(c, http.StatusNotFound, "Kelas tidak ditemukan")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Kelas dilepas dari daftarmu"})
}

// BackfillGuruKelas mengisi guru_kelas dari materi/tugas/kuis yang sudah ada, supaya Kelas Saya tidak kosong.
func BackfillGuruKelas() {
	if config.DB == nil {
		return
	}
	for _, tabel := range []string{"materis", "tugas", "kuis"} {
		config.DB.Exec("INSERT OR IGNORE INTO guru_kelas (guru_id, kelas_id, pelajaran_id, created_at) " +
			"SELECT DISTINCT guru_id, kelas_id, pelajaran_id, CURRENT_TIMESTAMP FROM " + tabel)
	}
}
