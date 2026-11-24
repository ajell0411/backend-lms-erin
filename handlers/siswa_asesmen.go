package handlers

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"e-class-backend/config"
	"e-class-backend/models"
)

type OpsiSoalSiswa struct {
	Kode string `json:"kode"`
	Teks string `json:"teks"`
}

type SoalSiswa struct {
	ID    uint            `json:"id"`
	Nomor int             `json:"nomor"`
	Teks  string          `json:"teks"`
	Jenis string          `json:"jenis"`
	Opsi  []OpsiSoalSiswa `json:"opsi"`
}

type soalInternal struct {
	ID    uint
	Nomor int
	Teks  string
	Jenis string
	Poin  int
	Opsi  []OpsiSoalSiswa
	Kunci string
}

type AsesmenSiswaItem struct {
	ID            uint               `json:"id"`
	Judul         string             `json:"judul"`
	Guru          string             `json:"guru,omitempty"`
	Deskripsi     string             `json:"deskripsi"`
	Tipe          string             `json:"tipe"` // "ujian_harian" | "kuis"
	Pelajaran     PelajaranSiswaItem `json:"pelajaran"`
	DurasiMenit   int                `json:"durasi_menit"`
	JadwalMulai   *string            `json:"jadwal_mulai,omitempty"`
	JadwalSelesai *string            `json:"jadwal_selesai,omitempty"`
	Status        string             `json:"status"` // "aktif" | "selesai" | "terkunci" | "berakhir"
	Nilai         *int               `json:"nilai,omitempty"`
	PerluDinilai  bool               `json:"perlu_dinilai"`
	NilaiMaks     int                `json:"nilai_maks"`
}

type DetailAsesmenSiswa struct {
	AsesmenSiswaItem
	Soal         []SoalSiswa `json:"soal"`
	WaktuSelesai *string     `json:"waktu_selesai,omitempty"`
	SudahDikirim bool        `json:"sudah_dikirim"`
}

func parseSoalJSON(raw string) []soalInternal {
	var list []soalInternal
	if strings.TrimSpace(raw) == "" {
		return list
	}

	var rawItems []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &rawItems); err != nil {
		// Jika format bukan array of object, buat 1 soal dari teks mentah
		list = append(list, soalInternal{
			ID:    1,
			Nomor: 1,
			Teks:  raw,
			Opsi: []OpsiSoalSiswa{
				{Kode: "A", Teks: "Benar"},
				{Kode: "B", Teks: "Salah"},
			},
			Kunci: "A",
		})
		return list
	}

	for idx, item := range rawItems {
		id := uint(idx + 1)
		if v, ok := item["id"].(float64); ok && v > 0 {
			id = uint(v)
		}
		nomor := idx + 1
		if v, ok := item["nomor"].(float64); ok && v > 0 {
			nomor = int(v)
		}

		teks := ""
		if v, ok := item["teks"].(string); ok {
			teks = v
		} else if v, ok := item["pertanyaan"].(string); ok {
			teks = v
		} else if v, ok := item["soal"].(string); ok {
			teks = v
		}

		var opsiList []OpsiSoalSiswa
		if rawOpsi, exists := item["opsi"]; exists {
			if arr, ok := rawOpsi.([]interface{}); ok {
				for oIdx, oVal := range arr {
					defaultKode := string(rune('A' + oIdx))
					if oMap, ok := oVal.(map[string]interface{}); ok {
						kode := defaultKode
						if k, ok := oMap["kode"].(string); ok && k != "" {
							kode = strings.ToUpper(k)
						}
						teksOpsi := ""
						if t, ok := oMap["teks"].(string); ok {
							teksOpsi = t
						}
						opsiList = append(opsiList, OpsiSoalSiswa{Kode: kode, Teks: teksOpsi})
					} else if oStr, ok := oVal.(string); ok {
						opsiList = append(opsiList, OpsiSoalSiswa{Kode: defaultKode, Teks: oStr})
					}
				}
			}
		}

		kunci := ""
		for _, keyName := range []string{"kunci", "kunci_jawaban", "kunciJawaban", "jawaban_benar", "jawabanBenar", "jawaban"} {
			if k, ok := item[keyName].(string); ok && strings.TrimSpace(k) != "" {
				kunci = strings.ToUpper(strings.TrimSpace(k))
				break
			}
		}
		jenis := "pilihan_ganda"
		if value, ok := item["jenis"].(string); ok && strings.EqualFold(strings.TrimSpace(value), "esai") {
			jenis = "esai"
		}
		poin := 1
		if value, ok := item["poin"].(float64); ok && value >= 1 {
			poin = int(value)
		}

		list = append(list, soalInternal{
			ID:    id,
			Nomor: nomor,
			Teks:  teks,
			Jenis: jenis,
			Poin:  poin,
			Opsi:  opsiList,
			Kunci: kunci,
		})
	}

	return list
}

func getAsesmenInfo(k models.Kuis, siswaID uint) (AsesmenSiswaItem, []soalInternal, *models.SiswaAsesmenSesi) {
	var meta models.AsesmenMetadata
	config.DB.Where("kuis_id = ?", k.ID).First(&meta)

	durasi := 60
	if meta.DurasiMenit > 0 {
		durasi = meta.DurasiMenit
	}

	var start time.Time
	if meta.JadwalMulai != nil {
		start = *meta.JadwalMulai
	} else if !k.CreatedAt.IsZero() {
		start = k.CreatedAt
	}

	var end time.Time
	if meta.JadwalSelesai != nil {
		end = *meta.JadwalSelesai
	} else if !k.Deadline.IsZero() {
		end = k.Deadline
	}

	tipe := "kuis"
	if k.Jenis == "ujian_harian" {
		tipe = "ujian_harian"
	}

	var sesi models.SiswaAsesmenSesi
	hasSesi := config.DB.Where("kuis_id = ? AND siswa_id = ?", k.ID, siswaID).First(&sesi).Error == nil

	var pengumpulan models.PengumpulanKuis
	hasPengumpulan := config.DB.Where("kuis_id = ? AND siswa_id = ?", k.ID, siswaID).First(&pengumpulan).Error == nil

	now := time.Now()
	status := "aktif"
	var nilai *int
	perluDinilai := false

	if (hasSesi && sesi.SudahDikirim) || hasPengumpulan {
		status = "selesai"
		jawaban := pengumpulan.Jawaban
		if hasSesi {
			jawaban = sesi.JawabanDraft
		}
		_, _, perluDinilai = guruHitungNilaiAsesmen(guruBacaSoal(k.Soal), guruBacaJawaban(jawaban), guruBacaNilaiEsai(jawaban))
		if hasSesi && sesi.Nilai != nil {
			nilai = sesi.Nilai
		} else if hasPengumpulan && pengumpulan.Nilai != nil {
			nilai = pengumpulan.Nilai
		}
	} else if !start.IsZero() && now.Before(start) {
		status = "terkunci"
	} else if !end.IsZero() && now.After(end) {
		status = "berakhir"
	} else {
		status = "aktif"
	}

	var jMulaiStr, jSelesaiStr *string
	if !start.IsZero() {
		s := start.Format(time.RFC3339)
		jMulaiStr = &s
	}
	if !end.IsZero() {
		s := end.Format(time.RFC3339)
		jSelesaiStr = &s
	}

	item := AsesmenSiswaItem{
		ID:        k.ID,
		Judul:     k.Judul,
		Guru:      k.Guru.Nama,
		Deskripsi: "Evaluasi pembelajaran untuk mengukur pemahaman materi.",
		Tipe:      tipe,
		Pelajaran: PelajaranSiswaItem{
			ID:   k.PelajaranID,
			Nama: k.Pelajaran.Nama,
			Kode: k.Pelajaran.Kode,
		},
		DurasiMenit:   durasi,
		JadwalMulai:   jMulaiStr,
		JadwalSelesai: jSelesaiStr,
		Status:        status,
		Nilai:         nilai,
		PerluDinilai:  perluDinilai,
		NilaiMaks:     100,
	}

	parsedSoal := parseSoalJSON(k.Soal)
	var pSesi *models.SiswaAsesmenSesi
	if hasSesi {
		pSesi = &sesi
	}
	return item, parsedSoal, pSesi
}

func ListSiswaAsesmen(c *gin.Context) {
	siswa, ok := getSiswaFromContext(c)
	if !ok {
		return
	}

	query := config.DB.Preload("Pelajaran").Where("kelas_id = ?", *siswa.KelasID)

	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(judul) LIKE ?", like)
	}

	pelajaranIDStr := c.Query("pelajaran_id")
	if pelajaranIDStr == "" {
		pelajaranIDStr = c.Query("pelajaranId")
	}
	if pelajaranIDStr != "" {
		if pid, err := strconv.Atoi(pelajaranIDStr); err == nil && pid > 0 {
			query = query.Where("pelajaran_id = ?", pid)
		}
	}

	if tipe := strings.TrimSpace(c.Query("tipe")); tipe != "" {
		query = query.Where("jenis = ?", tipe)
	}

	query = query.Preload("Guru")
	var kuisList []models.Kuis
	if err := query.Order("created_at DESC").Order("id DESC").Find(&kuisList).Error; err != nil {
		sendSiswaError(c, http.StatusInternalServerError, "Gagal mengambil daftar asesmen")
		return
	}

	items := make([]AsesmenSiswaItem, 0, len(kuisList))
	for _, k := range kuisList {
		info, _, _ := getAsesmenInfo(k, siswa.ID)
		items = append(items, info)
	}

	sendSiswaSuccess(c, http.StatusOK, "Berhasil memuat daftar asesmen", items)
}

func GetSiswaAsesmenByID(c *gin.Context) {
	siswa, ok := getSiswaFromContext(c)
	if !ok {
		return
	}

	id := c.Param("id")
	var kuis models.Kuis
	if err := config.DB.Preload("Pelajaran").Preload("Guru").First(&kuis, id).Error; err != nil {
		sendSiswaError(c, http.StatusNotFound, "Asesmen tidak ditemukan")
		return
	}

	if kuis.KelasID != *siswa.KelasID {
		sendSiswaError(c, http.StatusForbidden, "Asesmen ini bukan untuk kelas Anda")
		return
	}

	info, parsedSoal, sesi := getAsesmenInfo(kuis, siswa.ID)

	cleanSoal := make([]SoalSiswa, 0, len(parsedSoal))
	for _, s := range parsedSoal {
		cleanSoal = append(cleanSoal, SoalSiswa{
			ID:    s.ID,
			Nomor: s.Nomor,
			Teks:  s.Teks,
			Jenis: s.Jenis,
			Opsi:  s.Opsi,
		})
	}

	var waktuSelesaiStr *string
	sudahDikirim := info.Status == "selesai"
	if sesi != nil {
		s := sesi.WaktuSelesai.Format(time.RFC3339)
		waktuSelesaiStr = &s
		if sesi.SudahDikirim {
			sudahDikirim = true
		}
	}

	detail := DetailAsesmenSiswa{
		AsesmenSiswaItem: info,
		Soal:             cleanSoal,
		WaktuSelesai:     waktuSelesaiStr,
		SudahDikirim:     sudahDikirim,
	}

	sendSiswaSuccess(c, http.StatusOK, "Berhasil memuat detail asesmen", detail)
}

func MulaiSiswaAsesmen(c *gin.Context) {
	siswa, ok := getSiswaFromContext(c)
	if !ok {
		return
	}

	id := c.Param("id")
	var kuis models.Kuis
	if err := config.DB.First(&kuis, id).Error; err != nil {
		sendSiswaError(c, http.StatusNotFound, "Asesmen tidak ditemukan")
		return
	}

	if kuis.KelasID != *siswa.KelasID {
		sendSiswaError(c, http.StatusForbidden, "Asesmen ini bukan untuk kelas Anda")
		return
	}

	info, _, sesi := getAsesmenInfo(kuis, siswa.ID)
	now := time.Now()

	// Jika sudah dikirim
	if info.Status == "selesai" {
		waktuSelesai := now.Format(time.RFC3339)
		if sesi != nil {
			waktuSelesai = sesi.WaktuSelesai.Format(time.RFC3339)
		}
		sendSiswaSuccess(c, http.StatusOK, "Asesmen sudah diselesaikan", gin.H{
			"waktu_selesai": waktuSelesai,
			"sudah_dikirim": true,
		})
		return
	}

	// Cek jadwal
	if info.Status == "terkunci" || info.Status == "berakhir" {
		sendSiswaError(c, http.StatusForbidden, "Di luar jadwal pengerjaan asesmen")
		return
	}

	// Idempoten jika sesi sudah ada
	if sesi != nil {
		sendSiswaSuccess(c, http.StatusOK, "Asesmen sedang berjalan", gin.H{
			"waktu_selesai": sesi.WaktuSelesai.Format(time.RFC3339),
			"sudah_dikirim": sesi.SudahDikirim,
		})
		return
	}

	// Buat sesi baru
	waktuMulai := now
	waktuSelesai := waktuMulai.Add(time.Duration(info.DurasiMenit) * time.Minute)
	if !kuis.Deadline.IsZero() && waktuSelesai.After(kuis.Deadline) {
		waktuSelesai = kuis.Deadline
	}

	newSesi := models.SiswaAsesmenSesi{
		KuisID:       kuis.ID,
		SiswaID:      siswa.ID,
		WaktuMulai:   waktuMulai,
		WaktuSelesai: waktuSelesai,
		SudahDikirim: false,
		NilaiMaks:    100,
	}

	if err := config.DB.Create(&newSesi).Error; err != nil {
		sendSiswaError(c, http.StatusInternalServerError, "Gagal memulai asesmen")
		return
	}

	sendSiswaSuccess(c, http.StatusOK, "Asesmen berhasil dimulai", gin.H{
		"waktu_selesai": waktuSelesai.Format(time.RFC3339),
		"sudah_dikirim": false,
	})
}

func SimpanJawabanSiswaAsesmen(c *gin.Context) {
	siswa, ok := getSiswaFromContext(c)
	if !ok {
		return
	}

	id := c.Param("id")
	var kuis models.Kuis
	if err := config.DB.First(&kuis, id).Error; err != nil {
		sendSiswaError(c, http.StatusNotFound, "Asesmen tidak ditemukan")
		return
	}

	if kuis.KelasID != *siswa.KelasID {
		sendSiswaError(c, http.StatusForbidden, "Asesmen ini bukan untuk kelas Anda")
		return
	}

	var input struct {
		Jawaban map[string]string `json:"jawaban"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		sendSiswaError(c, http.StatusBadRequest, "Format jawaban tidak valid")
		return
	}

	var sesi models.SiswaAsesmenSesi
	if err := config.DB.Where("kuis_id = ? AND siswa_id = ?", kuis.ID, siswa.ID).First(&sesi).Error; err != nil {
		sendSiswaError(c, http.StatusBadRequest, "Sesi asesmen belum dimulai")
		return
	}

	if sesi.SudahDikirim {
		sendSiswaError(c, http.StatusConflict, "Asesmen sudah dikirim dan tidak dapat diubah lagi")
		return
	}

	bytes, _ := json.Marshal(input.Jawaban)
	sesi.JawabanDraft = string(bytes)
	if err := config.DB.Save(&sesi).Error; err != nil {
		sendSiswaError(c, http.StatusInternalServerError, "Jawaban draft gagal disimpan")
		return
	}

	sendSiswaSuccess(c, http.StatusOK, "Jawaban draft berhasil disimpan", nil)
}

func KirimSiswaAsesmen(c *gin.Context) {
	siswa, ok := getSiswaFromContext(c)
	if !ok {
		return
	}

	id := c.Param("id")
	var kuis models.Kuis
	if err := config.DB.First(&kuis, id).Error; err != nil {
		sendSiswaError(c, http.StatusNotFound, "Asesmen tidak ditemukan")
		return
	}

	if kuis.KelasID != *siswa.KelasID {
		sendSiswaError(c, http.StatusForbidden, "Asesmen ini bukan untuk kelas Anda")
		return
	}

	info, parsedSoal, sesi := getAsesmenInfo(kuis, siswa.ID)

	// Cek apakah sudah pernah dikirim -> 409
	if info.Status == "selesai" || (sesi != nil && sesi.SudahDikirim) {
		sendSiswaError(c, http.StatusConflict, "Asesmen sudah pernah dikirim sebelumnya")
		return
	}

	// Cek jadwal -> 403
	if info.Status == "terkunci" {
		sendSiswaError(c, http.StatusForbidden, "Asesmen belum dibuka untuk dikerjakan")
		return
	}

	var input struct {
		Jawaban map[string]string `json:"jawaban"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		sendSiswaError(c, http.StatusBadRequest, "Format jawaban tidak valid")
		return
	}

	jawabanMap := input.Jawaban
	if len(jawabanMap) == 0 && sesi != nil && sesi.JawabanDraft != "" {
		_ = json.Unmarshal([]byte(sesi.JawabanDraft), &jawabanMap)
	}
	if jawabanMap == nil {
		jawabanMap = make(map[string]string)
	}

	correct := 0
	totalPoin, didapatPoin := 0, 0
	perluDinilai := false
	for _, q := range parsedSoal {
		poin := q.Poin
		if poin < 1 {
			poin = 1
		}
		totalPoin += poin
		ans := ""
		if val, exists := jawabanMap[strconv.Itoa(int(q.ID))]; exists {
			ans = val
		} else if val, exists := jawabanMap[strconv.Itoa(q.Nomor)]; exists {
			ans = val
		}

		if q.Jenis == "esai" {
			perluDinilai = true
		} else if q.Kunci != "" && strings.EqualFold(strings.TrimSpace(ans), strings.TrimSpace(q.Kunci)) {
			didapatPoin += poin
			correct++
		}
	}

	var nilai *int
	if !perluDinilai {
		score := 100
		if totalPoin > 0 {
			score = int(math.Round(float64(didapatPoin) / float64(totalPoin) * 100.0))
		}
		nilai = &score
	}

	now := time.Now()
	jawabanBytes, _ := json.Marshal(map[string]interface{}{
		"jawaban": jawabanMap, "nilai_essay": map[string]int{},
	})
	jawabanStr := string(jawabanBytes)

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if sesi != nil {
			sesi.SudahDikirim = true
			sesi.DikirimAt = &now
			sesi.Nilai = nilai
			sesi.JawabanDraft = jawabanStr
			if err := tx.Save(sesi).Error; err != nil {
				return err
			}
		} else {
			waktuMulai := now.Add(-time.Duration(info.DurasiMenit) * time.Minute)
			sesi = &models.SiswaAsesmenSesi{
				KuisID: kuis.ID, SiswaID: siswa.ID, WaktuMulai: waktuMulai, WaktuSelesai: now,
				JawabanDraft: jawabanStr, SudahDikirim: true, DikirimAt: &now, Nilai: nilai, NilaiMaks: 100,
			}
			if err := tx.Create(sesi).Error; err != nil {
				return err
			}
		}

		var pengumpulan models.PengumpulanKuis
		err := tx.Where("kuis_id = ? AND siswa_id = ?", kuis.ID, siswa.ID).First(&pengumpulan).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			pengumpulan = models.PengumpulanKuis{KuisID: kuis.ID, SiswaID: siswa.ID, WaktuMulai: now}
		}
		pengumpulan.Jawaban = jawabanStr
		pengumpulan.Nilai = nilai
		pengumpulan.WaktuSelesai = now
		return tx.Save(&pengumpulan).Error
	})
	if err != nil {
		sendSiswaError(c, http.StatusInternalServerError, "Jawaban asesmen gagal disimpan")
		return
	}

	sendSiswaSuccess(c, http.StatusOK, "Asesmen berhasil dikirim", gin.H{
		"sudah_dikirim": true,
		"nilai":         nilai,
		"nilai_maks":    100,
		"perlu_dinilai": perluDinilai,
	})
}
