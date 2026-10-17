package handlers

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"e-class-backend/config"
	"e-class-backend/models"
)

// ===================== Statistik dashboard =====================

func GetPemantauanStatistik(c *gin.Context) {
	var guru, siswa, kelas, jurusan, pelajaran int64
	db := config.DB
	hasil := []error{
		db.Model(&models.User{}).Where("role = ?", "guru").Count(&guru).Error,
		db.Model(&models.User{}).Where("role = ?", "siswa").Count(&siswa).Error,
		db.Model(&models.Kelas{}).Count(&kelas).Error,
		db.Model(&models.Jurusan{}).Count(&jurusan).Error,
		db.Model(&models.Pelajaran{}).Count(&pelajaran).Error,
	}
	for _, err := range hasil {
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat statistik"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"total_guru":      guru,
		"total_siswa":     siswa,
		"total_kelas":     kelas,
		"total_jurusan":   jurusan,
		"total_pelajaran": pelajaran,
	}})
}

// ===================== Aktivitas guru =====================

type PemantauanItem struct {
	ID          uint       `json:"id"`
	Jenis       string     `json:"jenis"` // materi | tugas | ulangan
	GuruID      uint       `json:"guru_id"`
	GuruNama    string     `json:"guru_nama"`
	Judul       string     `json:"judul"`
	KelasID     uint       `json:"kelas_id"`
	Kelas       string     `json:"kelas"`
	PelajaranID uint       `json:"pelajaran_id"`
	Pelajaran   string     `json:"pelajaran"`
	DibuatPada  time.Time  `json:"dibuat_pada"`
	Deadline    *time.Time `json:"deadline"`
	Status      string     `json:"status"` // Aktif | Selesai, kosong untuk materi
}

type PemantauanDetail struct {
	PemantauanItem
	Deskripsi         string     `json:"deskripsi"`
	FileURL           string     `json:"file_url"`
	Instruksi         string     `json:"instruksi,omitempty"`
	Kriteria          string     `json:"kriteria,omitempty"`
	Bobot             *int       `json:"bobot,omitempty"`
	KategoriUlangan   string     `json:"kategori_ulangan,omitempty"`
	DurasiMenit       *int       `json:"durasi_menit,omitempty"`
	JadwalMulai       *time.Time `json:"jadwal_mulai,omitempty"`
	JadwalSelesai     *time.Time `json:"jadwal_selesai,omitempty"`
	JumlahSiswa       int64      `json:"jumlah_siswa"`
	JumlahPengumpulan *int64     `json:"jumlah_pengumpulan,omitempty"`
}

type pemantauanFilter struct {
	Jenis       string
	GuruID      uint
	KelasID     uint
	PelajaranID uint
	Dari        *time.Time
	Sampai      *time.Time // eksklusif
	Cari        string
	Limit       int
}

func pemantauanParseFilter(c *gin.Context) (pemantauanFilter, string) {
	var f pemantauanFilter

	f.Jenis = strings.ToLower(strings.TrimSpace(c.Query("jenis")))
	switch f.Jenis {
	case "", "semua":
		f.Jenis = ""
	case "materi", "tugas", "ulangan":
	default:
		return f, "Jenis harus materi, tugas, atau ulangan"
	}

	idParams := []struct {
		nama string
		dest *uint
	}{{"guru", &f.GuruID}, {"kelas", &f.KelasID}, {"pelajaran", &f.PelajaranID}}
	for _, p := range idParams {
		raw := strings.TrimSpace(c.Query(p.nama))
		if raw == "" {
			continue
		}
		n, err := strconv.ParseUint(raw, 10, 32)
		if err != nil || n == 0 {
			return f, "Filter " + p.nama + " harus berupa ID yang valid"
		}
		*p.dest = uint(n)
	}

	if raw := strings.TrimSpace(c.Query("tanggal_awal")); raw != "" {
		t, err := time.ParseInLocation("2006-01-02", raw, time.Local)
		if err != nil {
			return f, "Format tanggal_awal harus YYYY-MM-DD"
		}
		f.Dari = &t
	}
	if raw := strings.TrimSpace(c.Query("tanggal_akhir")); raw != "" {
		t, err := time.ParseInLocation("2006-01-02", raw, time.Local)
		if err != nil {
			return f, "Format tanggal_akhir harus YYYY-MM-DD"
		}
		akhir := t.AddDate(0, 0, 1)
		f.Sampai = &akhir
	}
	if f.Dari != nil && f.Sampai != nil && !f.Sampai.After(*f.Dari) {
		return f, "Tanggal akhir tidak boleh sebelum tanggal awal"
	}

	f.Cari = strings.TrimSpace(c.Query("q"))

	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return f, "Limit harus berupa angka positif"
		}
		if n > 500 {
			n = 500
		}
		f.Limit = n
	}
	return f, ""
}

func pemantauanApplyFilter(db *gorm.DB, f pemantauanFilter) *gorm.DB {
	if f.GuruID != 0 {
		db = db.Where("guru_id = ?", f.GuruID)
	}
	if f.KelasID != 0 {
		db = db.Where("kelas_id = ?", f.KelasID)
	}
	if f.PelajaranID != 0 {
		db = db.Where("pelajaran_id = ?", f.PelajaranID)
	}
	if f.Dari != nil {
		db = db.Where("created_at >= ?", *f.Dari)
	}
	if f.Sampai != nil {
		db = db.Where("created_at < ?", *f.Sampai)
	}
	if f.Cari != "" {
		db = db.Where("LOWER(judul) LIKE ?", "%"+strings.ToLower(f.Cari)+"%")
	}
	if f.Limit > 0 {
		db = db.Limit(f.Limit)
	}
	return db.Order("created_at DESC, id DESC")
}

func pemantauanDeadline(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func pemantauanStatus(deadline *time.Time, now time.Time) string {
	if deadline == nil {
		return ""
	}
	if now.After(*deadline) {
		return "Selesai"
	}
	return "Aktif"
}

func pemantauanItemMateri(r models.Materi) PemantauanItem {
	return PemantauanItem{
		ID: r.ID, Jenis: "materi",
		GuruID: r.GuruID, GuruNama: r.Guru.Nama,
		Judul:   r.Judul,
		KelasID: r.KelasID, Kelas: r.Kelas.Nama,
		PelajaranID: r.PelajaranID, Pelajaran: r.Pelajaran.Nama,
		DibuatPada: r.CreatedAt,
	}
}

func pemantauanItemTugas(r models.Tugas, now time.Time) PemantauanItem {
	deadline := pemantauanDeadline(r.Deadline)
	return PemantauanItem{
		ID: r.ID, Jenis: "tugas",
		GuruID: r.GuruID, GuruNama: r.Guru.Nama,
		Judul:   r.Judul,
		KelasID: r.KelasID, Kelas: r.Kelas.Nama,
		PelajaranID: r.PelajaranID, Pelajaran: r.Pelajaran.Nama,
		DibuatPada: r.CreatedAt,
		Deadline:   deadline,
		Status:     pemantauanStatus(deadline, now),
	}
}

func pemantauanItemUlangan(r models.Kuis, meta *models.AsesmenMetadata, now time.Time) PemantauanItem {
	deadline := pemantauanDeadline(r.Deadline)
	if meta != nil && meta.JadwalSelesai != nil {
		deadline = meta.JadwalSelesai
	}
	return PemantauanItem{
		ID: r.ID, Jenis: "ulangan",
		GuruID: r.GuruID, GuruNama: r.Guru.Nama,
		Judul:   r.Judul,
		KelasID: r.KelasID, Kelas: r.Kelas.Nama,
		PelajaranID: r.PelajaranID, Pelajaran: r.Pelajaran.Nama,
		DibuatPada: r.CreatedAt,
		Deadline:   deadline,
		Status:     pemantauanStatus(deadline, now),
	}
}

func pemantauanMateri(f pemantauanFilter) ([]PemantauanItem, error) {
	var rows []models.Materi
	if err := pemantauanApplyFilter(config.DB.Model(&models.Materi{}), f).
		Preload("Guru").Preload("Kelas").Preload("Pelajaran").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]PemantauanItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, pemantauanItemMateri(r))
	}
	return items, nil
}

func pemantauanTugas(f pemantauanFilter, now time.Time) ([]PemantauanItem, error) {
	var rows []models.Tugas
	if err := pemantauanApplyFilter(config.DB.Model(&models.Tugas{}), f).
		Preload("Guru").Preload("Kelas").Preload("Pelajaran").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]PemantauanItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, pemantauanItemTugas(r, now))
	}
	return items, nil
}

func pemantauanUlangan(f pemantauanFilter, now time.Time) ([]PemantauanItem, error) {
	var rows []models.Kuis
	if err := pemantauanApplyFilter(config.DB.Model(&models.Kuis{}), f).
		Preload("Guru").Preload("Kelas").Preload("Pelajaran").Find(&rows).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	metaByKuis := map[uint]models.AsesmenMetadata{}
	if len(ids) > 0 {
		var metas []models.AsesmenMetadata
		if err := config.DB.Where("kuis_id IN ?", ids).Find(&metas).Error; err != nil {
			return nil, err
		}
		for _, m := range metas {
			metaByKuis[m.KuisID] = m
		}
	}
	items := make([]PemantauanItem, 0, len(rows))
	for _, r := range rows {
		var mp *models.AsesmenMetadata
		if m, ok := metaByKuis[r.ID]; ok {
			salin := m
			mp = &salin
		}
		items = append(items, pemantauanItemUlangan(r, mp, now))
	}
	return items, nil
}

// GET /api/pemantauan/aktivitas-guru
// Query: jenis, guru, kelas, pelajaran (ID), tanggal_awal, tanggal_akhir (YYYY-MM-DD), q, limit
func ListPemantauanAktivitasGuru(c *gin.Context) {
	f, pesan := pemantauanParseFilter(c)
	if pesan != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": pesan})
		return
	}
	now := time.Now()
	items := make([]PemantauanItem, 0)

	if f.Jenis == "" || f.Jenis == "materi" {
		rows, err := pemantauanMateri(f)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat aktivitas materi"})
			return
		}
		items = append(items, rows...)
	}
	if f.Jenis == "" || f.Jenis == "tugas" {
		rows, err := pemantauanTugas(f, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat aktivitas tugas"})
			return
		}
		items = append(items, rows...)
	}
	if f.Jenis == "" || f.Jenis == "ulangan" {
		rows, err := pemantauanUlangan(f, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat aktivitas ulangan"})
			return
		}
		items = append(items, rows...)
	}

	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].DibuatPada.Equal(items[j].DibuatPada) {
			return items[i].DibuatPada.After(items[j].DibuatPada)
		}
		return items[i].ID > items[j].ID
	})
	if f.Limit > 0 && len(items) > f.Limit {
		items = items[:f.Limit]
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func pemantauanHitung(q *gorm.DB) (int64, error) {
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func pemantauanDetailError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aktivitas tidak ditemukan"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat detail aktivitas"})
}

// GET /api/pemantauan/aktivitas-guru/:jenis/:id
func GetPemantauanAktivitasGuruDetail(c *gin.Context) {
	jenis := strings.ToLower(c.Param("jenis"))
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID aktivitas tidak valid"})
		return
	}
	now := time.Now()
	var detail PemantauanDetail

	switch jenis {
	case "materi":
		var r models.Materi
		if err := config.DB.Preload("Guru").Preload("Kelas").Preload("Pelajaran").First(&r, id).Error; err != nil {
			pemantauanDetailError(c, err)
			return
		}
		detail.PemantauanItem = pemantauanItemMateri(r)
		detail.Deskripsi = r.Deskripsi
		detail.FileURL = r.FileUrl

	case "tugas":
		var r models.Tugas
		if err := config.DB.Preload("Guru").Preload("Kelas").Preload("Pelajaran").First(&r, id).Error; err != nil {
			pemantauanDetailError(c, err)
			return
		}
		detail.PemantauanItem = pemantauanItemTugas(r, now)
		detail.Deskripsi = r.Deskripsi
		detail.FileURL = r.LampiranUrl

		var meta models.TugasMetadata
		if err := config.DB.Where("tugas_id = ?", r.ID).First(&meta).Error; err == nil {
			bobot := meta.Bobot
			detail.Instruksi = meta.Instruksi
			detail.Kriteria = meta.Kriteria
			detail.Bobot = &bobot
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			pemantauanDetailError(c, err)
			return
		}
		kumpul, err := pemantauanHitung(config.DB.Model(&models.PengumpulanTugas{}).Where("tugas_id = ?", r.ID))
		if err != nil {
			pemantauanDetailError(c, err)
			return
		}
		detail.JumlahPengumpulan = &kumpul

	case "ulangan":
		var r models.Kuis
		if err := config.DB.Preload("Guru").Preload("Kelas").Preload("Pelajaran").First(&r, id).Error; err != nil {
			pemantauanDetailError(c, err)
			return
		}
		var meta models.AsesmenMetadata
		var mp *models.AsesmenMetadata
		if err := config.DB.Where("kuis_id = ?", r.ID).First(&meta).Error; err == nil {
			mp = &meta
			durasi, bobot := meta.DurasiMenit, meta.Bobot
			detail.DurasiMenit = &durasi
			detail.Bobot = &bobot
			detail.JadwalMulai = meta.JadwalMulai
			detail.JadwalSelesai = meta.JadwalSelesai
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			pemantauanDetailError(c, err)
			return
		}
		detail.PemantauanItem = pemantauanItemUlangan(r, mp, now)
		detail.KategoriUlangan = r.Jenis

		// Pengumpulan bisa tersimpan di sesi asesmen atau pengumpulan_kuis, ambil yang terbanyak.
		sesi, err := pemantauanHitung(config.DB.Model(&models.SiswaAsesmenSesi{}).Where("kuis_id = ? AND sudah_dikirim = ?", r.ID, true))
		if err != nil {
			pemantauanDetailError(c, err)
			return
		}
		lama, err := pemantauanHitung(config.DB.Model(&models.PengumpulanKuis{}).Where("kuis_id = ?", r.ID))
		if err != nil {
			pemantauanDetailError(c, err)
			return
		}
		kumpul := sesi
		if lama > kumpul {
			kumpul = lama
		}
		detail.JumlahPengumpulan = &kumpul

	default:
		c.JSON(http.StatusNotFound, gin.H{"error": "Jenis aktivitas tidak dikenal"})
		return
	}

	siswa, err := pemantauanHitung(config.DB.Model(&models.User{}).Where("role = ? AND kelas_id = ?", "siswa", detail.KelasID))
	if err != nil {
		pemantauanDetailError(c, err)
		return
	}
	detail.JumlahSiswa = siswa

	c.JSON(http.StatusOK, gin.H{"success": true, "data": detail})
}
