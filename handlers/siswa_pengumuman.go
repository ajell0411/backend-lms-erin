package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
	"e-class-backend/models"
)

type PengumumanSiswaItem struct {
	ID        string      `json:"id"`
	Jenis     string      `json:"jenis"` // "materi" | "tugas" | "asesmen"
	RefID     uint        `json:"ref_id"`
	Judul     string      `json:"judul"`
	Pelajaran interface{} `json:"pelajaran"`
	Guru      string      `json:"guru"`
	CreatedAt string      `json:"created_at"`
	Dilihat   bool        `json:"dilihat"`
	sortTime  time.Time   `json:"-"`
}

func getPelajaranNama(p models.Pelajaran) string {
	if p.Nama != "" {
		return p.Nama
	}
	return "Pelajaran"
}

func ListSiswaPengumuman(c *gin.Context) {
	siswa, ok := getSiswaFromContext(c)
	if !ok {
		return
	}

	jenisFilter := strings.ToLower(strings.TrimSpace(c.Query("jenis")))
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit < 1 {
		limit = 9
	}

	// Cek status baca siswa
	var bacaStatus models.SiswaPengumumanStatus
	_ = config.DB.Where("siswa_id = ?", siswa.ID).First(&bacaStatus).Error

	var readItems []models.PengumumanDibaca
	_ = config.DB.Where("siswa_id = ?", siswa.ID).Find(&readItems).Error
	readMap := make(map[string]bool)
	for _, r := range readItems {
		readMap[fmt.Sprintf("%s-%d", r.Jenis, r.RefID)] = true
	}

	isItemRead := func(jenis string, refID uint, createdAt time.Time) bool {
		if !bacaStatus.TerakhirDibaca.IsZero() && !createdAt.After(bacaStatus.TerakhirDibaca) {
			return true
		}
		if readMap[fmt.Sprintf("%s-%d", jenis, refID)] {
			return true
		}
		return false
	}

	var allItems []PengumumanSiswaItem

	// 1. Materi
	if jenisFilter == "" || jenisFilter == "materi" {
		var materis []models.Materi
		config.DB.Preload("Pelajaran").Preload("Guru").Where("kelas_id = ?", *siswa.KelasID).Find(&materis)
		for _, m := range materis {
			guruNama := m.Guru.Nama
			if guruNama == "" {
				guruNama = "Pengajar"
			}
			allItems = append(allItems, PengumumanSiswaItem{
				ID:        fmt.Sprintf("materi-%d", m.ID),
				Jenis:     "materi",
				RefID:     m.ID,
				Judul:     m.Judul,
				Pelajaran: getPelajaranNama(m.Pelajaran),
				Guru:      guruNama,
				CreatedAt: m.CreatedAt.Format(time.RFC3339),
				Dilihat:   isItemRead("materi", m.ID, m.CreatedAt),
				sortTime:  m.CreatedAt,
			})
		}
	}

	// 2. Tugas
	if jenisFilter == "" || jenisFilter == "tugas" {
		var tugasList []models.Tugas
		config.DB.Preload("Pelajaran").Preload("Guru").Where("kelas_id = ?", *siswa.KelasID).Find(&tugasList)
		for _, t := range tugasList {
			guruNama := t.Guru.Nama
			if guruNama == "" {
				guruNama = "Pengajar"
			}
			allItems = append(allItems, PengumumanSiswaItem{
				ID:        fmt.Sprintf("tugas-%d", t.ID),
				Jenis:     "tugas",
				RefID:     t.ID,
				Judul:     t.Judul,
				Pelajaran: getPelajaranNama(t.Pelajaran),
				Guru:      guruNama,
				CreatedAt: t.CreatedAt.Format(time.RFC3339),
				Dilihat:   isItemRead("tugas", t.ID, t.CreatedAt),
				sortTime:  t.CreatedAt,
			})
		}
	}

	// 3. Asesmen
	if jenisFilter == "" || jenisFilter == "asesmen" {
		var kuisList []models.Kuis
		config.DB.Preload("Pelajaran").Preload("Guru").Where("kelas_id = ?", *siswa.KelasID).Find(&kuisList)
		for _, k := range kuisList {
			guruNama := k.Guru.Nama
			if guruNama == "" {
				guruNama = "Pengajar"
			}
			allItems = append(allItems, PengumumanSiswaItem{
				ID:        fmt.Sprintf("asesmen-%d", k.ID),
				Jenis:     "asesmen",
				RefID:     k.ID,
				Judul:     k.Judul,
				Pelajaran: getPelajaranNama(k.Pelajaran),
				Guru:      guruNama,
				CreatedAt: k.CreatedAt.Format(time.RFC3339),
				Dilihat:   isItemRead("asesmen", k.ID, k.CreatedAt),
				sortTime:  k.CreatedAt,
			})
		}
	}

	// Sort urut dari terbaru
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].sortTime.After(allItems[j].sortTime)
	})

	total := len(allItems)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	pagedItems := allItems[start:end]
	hasMore := end < total

	sendSiswaSuccess(c, http.StatusOK, "Berhasil memuat pengumuman", gin.H{
		"items":    pagedItems,
		"total":    total,
		"has_more": hasMore,
	})
}

func TandaiSiswaPengumumanDibaca(c *gin.Context) {
	siswa, ok := getSiswaFromContext(c)
	if !ok {
		return
	}

	now := time.Now()
	var status models.SiswaPengumumanStatus
	if err := config.DB.Where("siswa_id = ?", siswa.ID).First(&status).Error; err == nil {
		status.TerakhirDibaca = now
		config.DB.Save(&status)
	} else {
		status = models.SiswaPengumumanStatus{
			SiswaID:        siswa.ID,
			TerakhirDibaca: now,
		}
		config.DB.Create(&status)
	}

	// Cek jika ada body spesifik item
	var body struct {
		Jenis string `json:"jenis"`
		RefID uint   `json:"ref_id"`
	}
	if err := c.ShouldBindJSON(&body); err == nil && body.Jenis != "" && body.RefID > 0 {
		var existing models.PengumumanDibaca
		if config.DB.Where("siswa_id = ? AND jenis = ? AND ref_id = ?", siswa.ID, body.Jenis, body.RefID).First(&existing).Error != nil {
			config.DB.Create(&models.PengumumanDibaca{
				SiswaID:  siswa.ID,
				Jenis:    body.Jenis,
				RefID:    body.RefID,
				DibacaAt: now,
			})
		}
	}

	sendSiswaSuccess(c, http.StatusOK, "Pengumuman berhasil ditandai sudah dibaca", gin.H{
		"dibaca_at": now.Format(time.RFC3339),
	})
}
