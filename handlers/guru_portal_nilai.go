package handlers

import (
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
	"e-class-backend/models"
)

func guruPredikat(n float64) string {
	switch {
	case n >= 90:
		return "A"
	case n >= 85:
		return "A-"
	case n >= 80:
		return "B+"
	case n >= 75:
		return "B"
	case n >= 70:
		return "B-"
	case n >= 60:
		return "C"
	case n >= 50:
		return "D"
	}
	return "E"
}

func guruKunciNilai(sid uint, jenis string, id uint) string {
	return strconv.Itoa(int(sid)) + ":" + jenis + strconv.Itoa(int(id))
}

// RekapGuruNilai: tabel nilai per kelas (kolom = tugas dan asesmen guru itu di kelas tersebut).
func RekapGuruNilai(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	gkID := guruAtoiUint(c.Query("guru_kelas_id"))
	if gkID == 0 {
		guruPortalGagal(c, http.StatusBadRequest, "Pilih kelas")
		return
	}
	var gk models.GuruKelas
	if err := config.DB.Preload("Kelas").Preload("Pelajaran").
		Where("id = ? AND guru_id = ?", gkID, guruID).First(&gk).Error; err != nil {
		guruPortalGagal(c, http.StatusNotFound, "Kelas tidak ditemukan")
		return
	}
	var siswa []models.User
	config.DB.Where("role = ? AND kelas_id = ?", "siswa", gk.KelasID).Order("nama ASC").Find(&siswa)
	var tugas []models.Tugas
	config.DB.Where("guru_id = ? AND kelas_id = ? AND pelajaran_id = ?", guruID, gk.KelasID, gk.PelajaranID).
		Order("created_at ASC").Find(&tugas)
	var kuis []models.Kuis
	config.DB.Where("guru_id = ? AND kelas_id = ? AND pelajaran_id = ?", guruID, gk.KelasID, gk.PelajaranID).
		Order("created_at ASC").Find(&kuis)

	kolom := []gin.H{}
	tugasIDs := []uint{}
	for _, t := range tugas {
		tugasIDs = append(tugasIDs, t.ID)
		kolom = append(kolom, gin.H{"key": "t" + strconv.Itoa(int(t.ID)), "jenis": "tugas", "judul": t.Judul})
	}
	kuisIDs := []uint{}
	for _, k := range kuis {
		kuisIDs = append(kuisIDs, k.ID)
		kolom = append(kolom, gin.H{"key": "k" + strconv.Itoa(int(k.ID)), "jenis": "asesmen", "judul": k.Judul})
	}

	nilaiMap := map[string]*int{}
	if len(tugasIDs) > 0 {
		var pt []models.PengumpulanTugas
		config.DB.Where("tugas_id IN ?", tugasIDs).Find(&pt)
		for _, p := range pt {
			nilaiMap[guruKunciNilai(p.SiswaID, "t", p.TugasID)] = p.Nilai
		}
	}
	if len(kuisIDs) > 0 {
		var sesi []models.SiswaAsesmenSesi
		config.DB.Where("kuis_id IN ? AND sudah_dikirim = ?", kuisIDs, true).Find(&sesi)
		for _, s := range sesi {
			nilaiMap[guruKunciNilai(s.SiswaID, "k", s.KuisID)] = s.Nilai
		}
	}

	rows := make([]gin.H, 0, len(siswa))
	for _, s := range siswa {
		nilai := gin.H{}
		total := 0.0
		jumlah := 0
		isi := func(key, jenis string, id uint) {
			if v, ada := nilaiMap[guruKunciNilai(s.ID, jenis, id)]; ada && v != nil {
				nilai[key] = *v
				total += float64(*v)
				jumlah++
				return
			}
			nilai[key] = nil
		}
		for _, t := range tugas {
			isi("t"+strconv.Itoa(int(t.ID)), "t", t.ID)
		}
		for _, k := range kuis {
			isi("k"+strconv.Itoa(int(k.ID)), "k", k.ID)
		}
		var akhir interface{}
		predikat := "-"
		if jumlah > 0 {
			r := math.Round(total/float64(jumlah)*10) / 10
			akhir = r
			predikat = guruPredikat(r)
		}
		rows = append(rows, gin.H{
			"id": s.ID, "nama": s.Nama, "nisn": guruPortalString(s.NISN),
			"nilai": nilai, "nilai_akhir": akhir, "predikat": predikat,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"kelas_nama": gk.Kelas.Nama, "pelajaran_nama": gk.Pelajaran.Nama,
		"kolom": kolom, "siswa": rows,
	}})
}

type guruAktivitasItem struct {
	Jenis         string    `json:"jenis"`
	SiswaID       uint      `json:"siswa_id"`
	SiswaNama     string    `json:"siswa_nama"`
	Judul         string    `json:"judul"`
	Waktu         time.Time `json:"waktu"`
	RefID         uint      `json:"ref_id"`
	PengumpulanID uint      `json:"pengumpulan_id,omitempty"`
	Terlambat     bool      `json:"terlambat"`
}

// ListGuruAktivitas: aktivitas siswa (mengumpulkan tugas / menyelesaikan asesmen) di kelas guru ini.
func ListGuruAktivitas(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	jenis := c.Query("jenis")
	db := config.DB
	items := []guruAktivitasItem{}

	if jenis == "" || jenis == "tugas" {
		var tugas []models.Tugas
		db.Where("guru_id = ?", guruID).Find(&tugas)
		ids := []uint{}
		judul := map[uint]string{}
		for _, t := range tugas {
			ids = append(ids, t.ID)
			judul[t.ID] = t.Judul
		}
		if len(ids) > 0 {
			var kumpul []models.PengumpulanTugas
			db.Where("tugas_id IN ?", ids).Order("waktu_submit DESC").Limit(limit).Find(&kumpul)
			for _, k := range kumpul {
				items = append(items, guruAktivitasItem{
					Jenis: "tugas", SiswaID: k.SiswaID, Judul: judul[k.TugasID], Waktu: k.WaktuSubmit,
					RefID: k.TugasID, PengumpulanID: k.ID, Terlambat: k.Terlambat,
				})
			}
		}
	}
	if jenis == "" || jenis == "asesmen" {
		var kuis []models.Kuis
		db.Where("guru_id = ?", guruID).Find(&kuis)
		ids := []uint{}
		judul := map[uint]string{}
		deadline := map[uint]time.Time{}
		for _, k := range kuis {
			ids = append(ids, k.ID)
			judul[k.ID] = k.Judul
			deadline[k.ID] = k.Deadline
		}
		if len(ids) > 0 {
			var sesi []models.SiswaAsesmenSesi
			db.Where("kuis_id IN ? AND sudah_dikirim = ?", ids, true).Order("dikirim_at DESC").Limit(limit).Find(&sesi)
			for _, s := range sesi {
				waktu := s.WaktuSelesai
				if s.DikirimAt != nil {
					waktu = *s.DikirimAt
				}
				items = append(items, guruAktivitasItem{
					Jenis: "asesmen", SiswaID: s.SiswaID, Judul: judul[s.KuisID], Waktu: waktu,
					RefID: s.KuisID, Terlambat: waktu.After(deadline[s.KuisID]),
				})
			}
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
	if len(items) > limit {
		items = items[:limit]
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}
