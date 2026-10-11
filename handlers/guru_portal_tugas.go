package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
	"e-class-backend/models"
)

var guruEkstTugas = []string{".pdf", ".doc", ".docx", ".ppt", ".pptx", ".xls", ".xlsx", ".zip", ".txt", ".png", ".jpg", ".jpeg"}

func guruParseWaktu(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func guruHitungTugas(t models.Tugas) (int64, int64) {
	var dik, sis int64
	config.DB.Model(&models.PengumpulanTugas{}).Where("tugas_id = ?", t.ID).Count(&dik)
	config.DB.Model(&models.User{}).Where("role = ? AND kelas_id = ?", "siswa", t.KelasID).Count(&sis)
	return dik, sis
}

func guruTugasJSON(t models.Tugas, dikumpulkan, siswa int64) gin.H {
	return gin.H{
		"id":             t.ID,
		"judul":          t.Judul,
		"deskripsi":      t.Deskripsi,
		"deadline":       t.Deadline,
		"aktif":          t.Deadline.After(time.Now()),
		"kelas_id":       t.KelasID,
		"kelas_nama":     t.Kelas.Nama,
		"pelajaran_id":   t.PelajaranID,
		"pelajaran_nama": t.Pelajaran.Nama,
		"lampiran_url":   t.LampiranUrl,
		"nama_lampiran":  guruNamaTampil(t.LampiranUrl),
		"dikumpulkan":    dikumpulkan,
		"jumlah_siswa":   siswa,
	}
}

func guruTugasMilik(c *gin.Context, guruID uint) (models.Tugas, bool) {
	var t models.Tugas
	id, ok := guruPortalParamID(c)
	if !ok {
		return t, false
	}
	if err := config.DB.Preload("Kelas").Preload("Pelajaran").
		Where("id = ? AND guru_id = ?", id, guruID).First(&t).Error; err != nil {
		guruPortalGagal(c, http.StatusNotFound, "Tugas tidak ditemukan")
		return t, false
	}
	return t, true
}

func ListGuruTugas(c *gin.Context) {
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
	var data []models.Tugas
	q.Order("created_at DESC").Find(&data)
	status := c.Query("status")
	now := time.Now()
	out := make([]gin.H, 0, len(data))
	for _, t := range data {
		aktif := t.Deadline.After(now)
		if (status == "aktif" && !aktif) || (status == "nonaktif" && aktif) {
			continue
		}
		dik, sis := guruHitungTugas(t)
		out = append(out, guruTugasJSON(t, dik, sis))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}

func TambahGuruTugas(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	judul := strings.TrimSpace(c.PostForm("judul"))
	kelasID := guruAtoiUint(c.PostForm("kelas_id"))
	pelID := guruAtoiUint(c.PostForm("pelajaran_id"))
	deadline, okW := guruParseWaktu(c.PostForm("deadline"))
	if judul == "" || kelasID == 0 || pelID == 0 || !okW {
		guruPortalGagal(c, http.StatusBadRequest, "Judul, kelas, dan batas waktu wajib diisi")
		return
	}
	if !guruPunyaKelas(guruID, kelasID, pelID) {
		guruPortalGagal(c, http.StatusForbidden, "Kelas ini belum ada di Kelas Saya")
		return
	}
	lampiran, _, err := guruSimpanFile(c, "lampiran", "tugas", guruEkstTugas, 20)
	if err != nil {
		guruPortalGagal(c, http.StatusBadRequest, err.Error())
		return
	}
	t := models.Tugas{
		Judul: judul, Deskripsi: strings.TrimSpace(c.PostForm("deskripsi")), LampiranUrl: lampiran,
		Deadline: deadline, PelajaranID: pelID, KelasID: kelasID, GuruID: guruID,
	}
	if err := config.DB.Create(&t).Error; err != nil {
		guruHapusFile(lampiran)
		guruPortalGagal(c, http.StatusInternalServerError, "Tugas gagal disimpan")
		return
	}
	config.DB.Preload("Kelas").Preload("Pelajaran").First(&t, t.ID)
	dik, sis := guruHitungTugas(t)
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": guruTugasJSON(t, dik, sis)})
}

func DetailGuruTugas(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	t, ok := guruTugasMilik(c, guruID)
	if !ok {
		return
	}
	var siswa []models.User
	config.DB.Where("role = ? AND kelas_id = ?", "siswa", t.KelasID).Order("nama ASC").Find(&siswa)
	var kumpul []models.PengumpulanTugas
	config.DB.Where("tugas_id = ?", t.ID).Find(&kumpul)
	byS := map[uint]models.PengumpulanTugas{}
	for _, k := range kumpul {
		byS[k.SiswaID] = k
	}
	rows := make([]gin.H, 0, len(siswa))
	for _, s := range siswa {
		row := gin.H{
			"siswa_id": s.ID, "nama": s.Nama, "nisn": guruPortalString(s.NISN), "foto_url": s.FotoURL,
			"status": "belum", "waktu_kumpul": nil, "terlambat": false, "nilai": nil, "pengumpulan_id": nil,
		}
		if k, ada := byS[s.ID]; ada {
			row["status"] = "dikumpulkan"
			row["waktu_kumpul"] = k.WaktuSubmit
			row["terlambat"] = k.Terlambat
			row["nilai"] = k.Nilai
			row["pengumpulan_id"] = k.ID
		}
		rows = append(rows, row)
	}
	data := guruTugasJSON(t, int64(len(kumpul)), int64(len(siswa)))
	data["siswa"] = rows
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func UbahGuruTugas(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	t, ok := guruTugasMilik(c, guruID)
	if !ok {
		return
	}
	update := map[string]interface{}{}
	if v, ada := c.GetPostForm("judul"); ada {
		v = strings.TrimSpace(v)
		if v == "" {
			guruPortalGagal(c, http.StatusBadRequest, "Judul tidak boleh kosong")
			return
		}
		update["judul"] = v
	}
	if v, ada := c.GetPostForm("deskripsi"); ada {
		update["deskripsi"] = strings.TrimSpace(v)
	}
	if v, ada := c.GetPostForm("deadline"); ada && strings.TrimSpace(v) != "" {
		d, okW := guruParseWaktu(v)
		if !okW {
			guruPortalGagal(c, http.StatusBadRequest, "Format batas waktu tidak valid")
			return
		}
		update["deadline"] = d
	}
	kelasID, pelID := t.KelasID, t.PelajaranID
	if v := guruAtoiUint(c.PostForm("kelas_id")); v > 0 {
		kelasID = v
	}
	if v := guruAtoiUint(c.PostForm("pelajaran_id")); v > 0 {
		pelID = v
	}
	if kelasID != t.KelasID || pelID != t.PelajaranID {
		if !guruPunyaKelas(guruID, kelasID, pelID) {
			guruPortalGagal(c, http.StatusForbidden, "Kelas ini belum ada di Kelas Saya")
			return
		}
		update["kelas_id"] = kelasID
		update["pelajaran_id"] = pelID
	}
	lampiran, ada, err := guruSimpanFile(c, "lampiran", "tugas", guruEkstTugas, 20)
	if err != nil {
		guruPortalGagal(c, http.StatusBadRequest, err.Error())
		return
	}
	lama := t.LampiranUrl
	if ada {
		update["lampiran_url"] = lampiran
	}
	if len(update) > 0 {
		config.DB.Model(&models.Tugas{}).Where("id = ?", t.ID).Updates(update)
	}
	if ada && lama != "" && lama != lampiran {
		guruHapusFile(lama)
	}
	config.DB.Preload("Kelas").Preload("Pelajaran").First(&t, t.ID)
	dik, sis := guruHitungTugas(t)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": guruTugasJSON(t, dik, sis)})
}

// NonaktifkanGuruTugas menutup tugas dengan memajukan batas waktu ke sekarang.
// Untuk membuka lagi, ubah batas waktunya lewat Edit.
func NonaktifkanGuruTugas(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	t, ok := guruTugasMilik(c, guruID)
	if !ok {
		return
	}
	config.DB.Model(&models.Tugas{}).Where("id = ?", t.ID).Update("deadline", time.Now())
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Tugas dinonaktifkan"})
}

func HapusGuruTugas(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	t, ok := guruTugasMilik(c, guruID)
	if !ok {
		return
	}
	config.DB.Where("tugas_id = ?", t.ID).Delete(&models.PengumpulanTugas{})
	config.DB.Where("tugas_id = ?", t.ID).Delete(&models.TugasMetadata{})
	config.DB.Where("tugas_id = ?", t.ID).Delete(&models.SiswaPengumpulanCatatan{})
	config.DB.Delete(&models.Tugas{}, t.ID)
	guruHapusFile(t.LampiranUrl)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Tugas dihapus"})
}

func guruPengumpulanMilik(c *gin.Context, guruID uint) (models.PengumpulanTugas, models.Tugas, bool) {
	var p models.PengumpulanTugas
	var t models.Tugas
	id, ok := guruPortalParamID(c)
	if !ok {
		return p, t, false
	}
	if err := config.DB.First(&p, id).Error; err != nil {
		guruPortalGagal(c, http.StatusNotFound, "Pengumpulan tidak ditemukan")
		return p, t, false
	}
	if err := config.DB.Preload("Kelas").Preload("Pelajaran").
		Where("id = ? AND guru_id = ?", p.TugasID, guruID).First(&t).Error; err != nil {
		guruPortalGagal(c, http.StatusNotFound, "Pengumpulan tidak ditemukan")
		return p, t, false
	}
	return p, t, true
}

func DetailGuruPengumpulan(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	p, t, ok := guruPengumpulanMilik(c, guruID)
	if !ok {
		return
	}
	var siswa models.User
	config.DB.First(&siswa, p.SiswaID)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"id":             p.ID,
		"tugas_id":       t.ID,
		"tugas_judul":    t.Judul,
		"kelas_nama":     t.Kelas.Nama,
		"pelajaran_nama": t.Pelajaran.Nama,
		"siswa_id":       siswa.ID,
		"siswa_nama":     siswa.Nama,
		"siswa_nisn":     guruPortalString(siswa.NISN),
		"siswa_foto":     siswa.FotoURL,
		"waktu_submit":   p.WaktuSubmit,
		"terlambat":      p.Terlambat,
		"nilai":          p.Nilai,
		"feedback":       p.Feedback,
		"nama_file":      guruNamaTampil(p.FileJawabanUrl),
		"ada_file":       guruFileAda(p.FileJawabanUrl),
	}})
}

func FileGuruPengumpulan(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	p, _, ok := guruPengumpulanMilik(c, guruID)
	if !ok {
		return
	}
	guruKirimFile(c, p.FileJawabanUrl)
}

func NilaiGuruPengumpulan(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	p, _, ok := guruPengumpulanMilik(c, guruID)
	if !ok {
		return
	}
	var in struct {
		Nilai    *int   `json:"nilai"`
		Feedback string `json:"feedback"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Nilai == nil {
		guruPortalGagal(c, http.StatusBadRequest, "Nilai wajib diisi (0-100)")
		return
	}
	if *in.Nilai < 0 || *in.Nilai > 100 {
		guruPortalGagal(c, http.StatusBadRequest, "Nilai harus antara 0 dan 100")
		return
	}
	config.DB.Model(&models.PengumpulanTugas{}).Where("id = ?", p.ID).Updates(map[string]interface{}{
		"nilai": *in.Nilai, "feedback": strings.TrimSpace(in.Feedback), "status_penilaian": "dinilai",
	})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Nilai disimpan", "data": gin.H{"nilai": *in.Nilai}})
}
