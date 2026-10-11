package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
	"e-class-backend/models"
)

var guruEkstMateri = []string{".pdf", ".doc", ".docx", ".ppt", ".pptx", ".xls", ".xlsx"}

func guruPunyaKelas(guruID, kelasID, pelajaranID uint) bool {
	var n int64
	config.DB.Model(&models.GuruKelas{}).
		Where("guru_id = ? AND kelas_id = ? AND pelajaran_id = ?", guruID, kelasID, pelajaranID).Count(&n)
	return n > 0
}

func guruMateriJSON(m models.Materi) gin.H {
	return gin.H{
		"id":             m.ID,
		"judul":          m.Judul,
		"deskripsi":      m.Deskripsi,
		"kelas_id":       m.KelasID,
		"kelas_nama":     m.Kelas.Nama,
		"pelajaran_id":   m.PelajaranID,
		"pelajaran_nama": m.Pelajaran.Nama,
		"file_url":       m.FileUrl,
		"nama_file":      guruNamaTampil(m.FileUrl),
		"ada_file":       guruFileAda(m.FileUrl),
		"created_at":     m.CreatedAt,
	}
}

func guruMateriMilik(c *gin.Context, guruID uint) (models.Materi, bool) {
	var m models.Materi
	id, ok := guruPortalParamID(c)
	if !ok {
		return m, false
	}
	if err := config.DB.Preload("Kelas").Preload("Pelajaran").
		Where("id = ? AND guru_id = ?", id, guruID).First(&m).Error; err != nil {
		guruPortalGagal(c, http.StatusNotFound, "Materi tidak ditemukan")
		return m, false
	}
	return m, true
}

func ListGuruMateri(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	q := config.DB.Preload("Kelas").Preload("Pelajaran").Where("guru_id = ?", guruID)
	if kid := guruAtoiUint(c.Query("kelas_id")); kid > 0 {
		q = q.Where("kelas_id = ?", kid)
	}
	if s := strings.ToLower(strings.TrimSpace(c.Query("q"))); s != "" {
		q = q.Where("LOWER(judul) LIKE ?", "%"+s+"%")
	}
	var data []models.Materi
	q.Order("created_at DESC").Find(&data)
	out := make([]gin.H, 0, len(data))
	for _, m := range data {
		out = append(out, guruMateriJSON(m))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}

func TambahGuruMateri(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	judul := strings.TrimSpace(c.PostForm("judul"))
	kelasID := guruAtoiUint(c.PostForm("kelas_id"))
	pelID := guruAtoiUint(c.PostForm("pelajaran_id"))
	if judul == "" || kelasID == 0 || pelID == 0 {
		guruPortalGagal(c, http.StatusBadRequest, "Nama materi dan kelas wajib diisi")
		return
	}
	if !guruPunyaKelas(guruID, kelasID, pelID) {
		guruPortalGagal(c, http.StatusForbidden, "Kelas ini belum ada di Kelas Saya")
		return
	}
	fileURL, ada, err := guruSimpanFile(c, "file", "materi", guruEkstMateri, 50)
	if err != nil {
		guruPortalGagal(c, http.StatusBadRequest, err.Error())
		return
	}
	if !ada {
		guruPortalGagal(c, http.StatusBadRequest, "File materi wajib diunggah")
		return
	}
	m := models.Materi{
		Judul: judul, Deskripsi: strings.TrimSpace(c.PostForm("deskripsi")), FileUrl: fileURL,
		PelajaranID: pelID, KelasID: kelasID, GuruID: guruID,
	}
	if err := config.DB.Create(&m).Error; err != nil {
		guruHapusFile(fileURL)
		guruPortalGagal(c, http.StatusInternalServerError, "Materi gagal disimpan")
		return
	}
	config.DB.Preload("Kelas").Preload("Pelajaran").First(&m, m.ID)
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": guruMateriJSON(m)})
}

func UbahGuruMateri(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	m, ok := guruMateriMilik(c, guruID)
	if !ok {
		return
	}
	update := map[string]interface{}{}
	if v, ada := c.GetPostForm("judul"); ada {
		v = strings.TrimSpace(v)
		if v == "" {
			guruPortalGagal(c, http.StatusBadRequest, "Nama materi tidak boleh kosong")
			return
		}
		update["judul"] = v
	}
	if v, ada := c.GetPostForm("deskripsi"); ada {
		update["deskripsi"] = strings.TrimSpace(v)
	}
	kelasID, pelID := m.KelasID, m.PelajaranID
	if v := guruAtoiUint(c.PostForm("kelas_id")); v > 0 {
		kelasID = v
	}
	if v := guruAtoiUint(c.PostForm("pelajaran_id")); v > 0 {
		pelID = v
	}
	if kelasID != m.KelasID || pelID != m.PelajaranID {
		if !guruPunyaKelas(guruID, kelasID, pelID) {
			guruPortalGagal(c, http.StatusForbidden, "Kelas ini belum ada di Kelas Saya")
			return
		}
		update["kelas_id"] = kelasID
		update["pelajaran_id"] = pelID
	}
	fileURL, ada, err := guruSimpanFile(c, "file", "materi", guruEkstMateri, 50)
	if err != nil {
		guruPortalGagal(c, http.StatusBadRequest, err.Error())
		return
	}
	lama := m.FileUrl
	if ada {
		update["file_url"] = fileURL
	}
	if len(update) > 0 {
		config.DB.Model(&models.Materi{}).Where("id = ?", m.ID).Updates(update)
	}
	if ada && lama != "" && lama != fileURL {
		guruHapusFile(lama)
	}
	config.DB.Preload("Kelas").Preload("Pelajaran").First(&m, m.ID)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": guruMateriJSON(m)})
}

func HapusGuruMateri(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	m, ok := guruMateriMilik(c, guruID)
	if !ok {
		return
	}
	config.DB.Delete(&models.Materi{}, m.ID)
	guruHapusFile(m.FileUrl)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Materi dihapus"})
}

func UnduhGuruMateri(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	m, ok := guruMateriMilik(c, guruID)
	if !ok {
		return
	}
	guruKirimFile(c, m.FileUrl)
}
