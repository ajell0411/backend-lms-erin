package handlers

import (
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
	"e-class-backend/models"
)

var guruPortalHari = []string{"minggu", "senin", "selasa", "rabu", "kamis", "jumat", "sabtu"}

type guruPortalAktivitas struct {
	Jenis     string    `json:"jenis"` // tugas | asesmen
	SiswaID   uint      `json:"siswa_id"`
	SiswaNama string    `json:"siswa_nama"`
	Judul     string    `json:"judul"`
	Waktu     time.Time `json:"waktu"`
}

func GetGuruDashboard(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	db := config.DB
	now := time.Now()

	var guru models.User
	db.First(&guru, guruID)

	var jumlahKelas int64
	db.Model(&models.GuruKelas{}).Where("guru_id = ?", guruID).Distinct("kelas_id").Count(&jumlahKelas)

	var tugasList []models.Tugas
	db.Where("guru_id = ?", guruID).Find(&tugasList)
	var kuisList []models.Kuis
	db.Where("guru_id = ?", guruID).Find(&kuisList)

	tugasAktif, asesmenAktif := 0, 0
	tugasIDs := []uint{}
	tugasJudul := map[uint]string{}
	for _, t := range tugasList {
		tugasIDs = append(tugasIDs, t.ID)
		tugasJudul[t.ID] = t.Judul
		if t.Deadline.After(now) {
			tugasAktif++
		}
	}
	kuisIDs := []uint{}
	kuisJudul := map[uint]string{}
	for _, k := range kuisList {
		kuisIDs = append(kuisIDs, k.ID)
		kuisJudul[k.ID] = k.Judul
		if k.Deadline.After(now) {
			asesmenAktif++
		}
	}

	var perluDinilai int64
	items := []guruPortalAktivitas{}

	if len(tugasIDs) > 0 {
		db.Model(&models.PengumpulanTugas{}).Where("tugas_id IN ? AND nilai IS NULL", tugasIDs).Count(&perluDinilai)
		var kumpul []models.PengumpulanTugas
		db.Where("tugas_id IN ?", tugasIDs).Order("waktu_submit DESC").Limit(10).Find(&kumpul)
		for _, k := range kumpul {
			items = append(items, guruPortalAktivitas{
				Jenis: "tugas", SiswaID: k.SiswaID, Judul: tugasJudul[k.TugasID], Waktu: k.WaktuSubmit,
			})
		}
	}
	if len(kuisIDs) > 0 {
		var sesi []models.SiswaAsesmenSesi
		db.Where("kuis_id IN ? AND sudah_dikirim = ?", kuisIDs, true).Order("dikirim_at DESC").Limit(10).Find(&sesi)
		for _, s := range sesi {
			waktu := s.WaktuSelesai
			if s.DikirimAt != nil {
				waktu = *s.DikirimAt
			}
			items = append(items, guruPortalAktivitas{
				Jenis: "asesmen", SiswaID: s.SiswaID, Judul: kuisJudul[s.KuisID], Waktu: waktu,
			})
		}
	}

	if len(items) > 0 {
		ids := []uint{}
		seen := map[uint]bool{}
		for _, it := range items {
			if !seen[it.SiswaID] {
				seen[it.SiswaID] = true
				ids = append(ids, it.SiswaID)
			}
		}
		var users []models.User
		db.Where("id IN ?", ids).Find(&users)
		nama := map[uint]string{}
		for _, u := range users {
			nama[u.ID] = u.Nama
		}
		for i := range items {
			items[i].SiswaNama = nama[items[i].SiswaID]
		}
	}
	sort.Slice(items, func(a, b int) bool { return items[a].Waktu.After(items[b].Waktu) })
	if len(items) > 10 {
		items = items[:10]
	}

	var jadwal []models.Jadwal
	db.Preload("Kelas").Preload("Pelajaran").
		Where("guru_id = ? AND hari = ?", guruID, guruPortalHari[int(now.Weekday())]).
		Order("jam_mulai ASC").Find(&jadwal)
	jadwalOut := make([]gin.H, 0, len(jadwal))
	for _, j := range jadwal {
		jadwalOut = append(jadwalOut, guruPortalJadwalJSON(j))
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"nama":            guru.Nama,
		"jumlah_kelas":    jumlahKelas,
		"tugas_aktif":     tugasAktif,
		"asesmen_aktif":   asesmenAktif,
		"perlu_dinilai":   perluDinilai,
		"jadwal_hari_ini": jadwalOut,
		"aktivitas":       items,
	}})
}
