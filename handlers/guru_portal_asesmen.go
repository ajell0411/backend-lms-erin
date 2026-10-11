package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
	"e-class-backend/models"
)

const guruKKM = 75

type guruOpsi struct {
	Kode string `json:"kode"`
	Teks string `json:"teks"`
}

type guruSoal struct {
	ID         int        `json:"id"`
	Nomor      int        `json:"nomor"`
	Teks       string     `json:"teks"`
	Opsi       []guruOpsi `json:"opsi"`
	Kunci      string     `json:"kunci"`
	Poin       int        `json:"poin,omitempty"`
	Pembahasan string     `json:"pembahasan,omitempty"`
}

type guruSoalInput struct {
	Teks       string     `json:"teks"`
	Poin       int        `json:"poin"`
	Opsi       []guruOpsi `json:"opsi"`
	Kunci      string     `json:"kunci"`
	Pembahasan string     `json:"pembahasan"`
}

type guruAsesmenInput struct {
	Judul       string `json:"judul"`
	Jenis       string `json:"jenis"`
	KelasID     uint   `json:"kelas_id"`
	PelajaranID uint   `json:"pelajaran_id"`
	Tanggal     string `json:"tanggal"`
	JamMulai    string `json:"jam_mulai"`
	JamSelesai  string `json:"jam_selesai"`
}

func guruBacaSoal(raw string) []guruSoal {
	out := []guruSoal{}
	if strings.TrimSpace(raw) == "" {
		return out
	}
	var tmp []guruSoal
	if err := json.Unmarshal([]byte(raw), &tmp); err != nil {
		return out
	}
	if tmp != nil {
		out = tmp
	}
	return out
}

func guruSimpanSoal(kuisID uint, soal []guruSoal) error {
	for i := range soal {
		soal[i].Nomor = i + 1
	}
	b, err := json.Marshal(soal)
	if err != nil {
		return err
	}
	return config.DB.Model(&models.Kuis{}).Where("id = ?", kuisID).Update("soal", string(b)).Error
}

func guruAnyString(v interface{}) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case map[string]interface{}:
		for _, key := range []string{"kode", "jawaban", "pilihan"} {
			if s, ok := x[key].(string); ok {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func guruAnyInt(v interface{}) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(x))
		return n
	}
	return 0
}

// guruBacaJawaban membaca jawaban siswa dengan toleran terhadap format JSON (objek atau array).
func guruBacaJawaban(raw string) map[int]string {
	hasil := map[int]string{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return hasil
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &obj); err == nil {
		if inner, ada := obj["jawaban"].(map[string]interface{}); ada {
			obj = inner
		}
		for k, v := range obj {
			if n, err := strconv.Atoi(k); err == nil {
				if s := guruAnyString(v); s != "" {
					hasil[n] = s
				}
			}
		}
		return hasil
	}
	var arr []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &arr); err == nil {
		for _, it := range arr {
			id := 0
			for _, key := range []string{"soal_id", "soalId", "id", "nomor"} {
				if v, ada := it[key]; ada {
					if n := guruAnyInt(v); n > 0 {
						id = n
						break
					}
				}
			}
			jw := ""
			for _, key := range []string{"jawaban", "kode", "pilihan", "opsi"} {
				if v, ada := it[key]; ada {
					if s := guruAnyString(v); s != "" {
						jw = s
						break
					}
				}
			}
			if id > 0 && jw != "" {
				hasil[id] = jw
			}
		}
	}
	return hasil
}

func guruWaktuAsesmen(in *guruAsesmenInput) (time.Time, time.Time, string) {
	in.Judul = strings.TrimSpace(in.Judul)
	if in.Jenis != "ujian_harian" {
		in.Jenis = "kuis"
	}
	if in.Judul == "" || in.KelasID == 0 || in.PelajaranID == 0 {
		return time.Time{}, time.Time{}, "Judul dan kelas wajib diisi"
	}
	mulai, ok1 := guruParseWaktu(in.Tanggal + "T" + in.JamMulai)
	selesai, ok2 := guruParseWaktu(in.Tanggal + "T" + in.JamSelesai)
	if !ok1 || !ok2 {
		return time.Time{}, time.Time{}, "Tanggal dan jam wajib diisi"
	}
	if !selesai.After(mulai) {
		return time.Time{}, time.Time{}, "Jam selesai harus setelah jam mulai"
	}
	return mulai, selesai, ""
}

func guruSimpanMetaAsesmen(kuisID uint, mulai, selesai time.Time) {
	dur := int(selesai.Sub(mulai).Minutes())
	var meta models.AsesmenMetadata
	config.DB.Where("kuis_id = ?", kuisID).Limit(1).Find(&meta)
	if meta.ID == 0 {
		meta = models.AsesmenMetadata{KuisID: kuisID, DurasiMenit: dur, JadwalMulai: &mulai, JadwalSelesai: &selesai, Bobot: 100}
		config.DB.Create(&meta)
		return
	}
	config.DB.Model(&models.AsesmenMetadata{}).Where("id = ?", meta.ID).Updates(map[string]interface{}{
		"durasi_menit": dur, "jadwal_mulai": mulai, "jadwal_selesai": selesai,
	})
}

func guruMetaAsesmen(kuisID uint) *models.AsesmenMetadata {
	var meta models.AsesmenMetadata
	config.DB.Where("kuis_id = ?", kuisID).Limit(1).Find(&meta)
	if meta.ID == 0 {
		return nil
	}
	return &meta
}

func guruHitungAsesmen(k models.Kuis) (int64, int64) {
	var selesai, siswa int64
	config.DB.Model(&models.SiswaAsesmenSesi{}).Where("kuis_id = ? AND sudah_dikirim = ?", k.ID, true).Count(&selesai)
	config.DB.Model(&models.User{}).Where("role = ? AND kelas_id = ?", "siswa", k.KelasID).Count(&siswa)
	return selesai, siswa
}

func guruAsesmenJSON(k models.Kuis) gin.H {
	meta := guruMetaAsesmen(k.ID)
	selesai, siswa := guruHitungAsesmen(k)
	now := time.Now()
	status := "aktif"
	if now.After(k.Deadline) {
		status = "berakhir"
	} else if meta != nil && meta.JadwalMulai != nil && now.Before(*meta.JadwalMulai) {
		status = "terjadwal"
	}
	var mulai interface{}
	durasi := 0
	if meta != nil {
		durasi = meta.DurasiMenit
		if meta.JadwalMulai != nil {
			mulai = *meta.JadwalMulai
		}
	}
	return gin.H{
		"id":             k.ID,
		"judul":          k.Judul,
		"jenis":          k.Jenis,
		"kelas_id":       k.KelasID,
		"kelas_nama":     k.Kelas.Nama,
		"pelajaran_id":   k.PelajaranID,
		"pelajaran_nama": k.Pelajaran.Nama,
		"deadline":       k.Deadline,
		"jadwal_mulai":   mulai,
		"durasi_menit":   durasi,
		"status":         status,
		"jumlah_soal":    len(guruBacaSoal(k.Soal)),
		"selesai":        selesai,
		"jumlah_siswa":   siswa,
	}
}

func guruKuisMilik(c *gin.Context, guruID uint) (models.Kuis, bool) {
	var k models.Kuis
	id, ok := guruPortalParamID(c)
	if !ok {
		return k, false
	}
	if err := config.DB.Preload("Kelas").Preload("Pelajaran").
		Where("id = ? AND guru_id = ?", id, guruID).First(&k).Error; err != nil {
		guruPortalGagal(c, http.StatusNotFound, "Asesmen tidak ditemukan")
		return k, false
	}
	return k, true
}

func ListGuruAsesmen(c *gin.Context) {
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
	var data []models.Kuis
	q.Order("created_at DESC").Find(&data)
	out := make([]gin.H, 0, len(data))
	for _, k := range data {
		out = append(out, guruAsesmenJSON(k))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}

func TambahGuruAsesmen(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	var in guruAsesmenInput
	if err := c.ShouldBindJSON(&in); err != nil {
		guruPortalGagal(c, http.StatusBadRequest, "Data tidak valid")
		return
	}
	mulai, selesai, pesan := guruWaktuAsesmen(&in)
	if pesan != "" {
		guruPortalGagal(c, http.StatusBadRequest, pesan)
		return
	}
	if !guruPunyaKelas(guruID, in.KelasID, in.PelajaranID) {
		guruPortalGagal(c, http.StatusForbidden, "Kelas ini belum ada di Kelas Saya")
		return
	}
	k := models.Kuis{
		Judul: in.Judul, Jenis: in.Jenis, GuruID: guruID, PelajaranID: in.PelajaranID,
		KelasID: in.KelasID, Soal: "[]", Deadline: selesai,
	}
	if err := config.DB.Create(&k).Error; err != nil {
		guruPortalGagal(c, http.StatusInternalServerError, "Asesmen gagal disimpan")
		return
	}
	guruSimpanMetaAsesmen(k.ID, mulai, selesai)
	config.DB.Preload("Kelas").Preload("Pelajaran").First(&k, k.ID)
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": guruAsesmenJSON(k)})
}

func UbahGuruAsesmen(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	k, ok := guruKuisMilik(c, guruID)
	if !ok {
		return
	}
	var in guruAsesmenInput
	if err := c.ShouldBindJSON(&in); err != nil {
		guruPortalGagal(c, http.StatusBadRequest, "Data tidak valid")
		return
	}
	mulai, selesai, pesan := guruWaktuAsesmen(&in)
	if pesan != "" {
		guruPortalGagal(c, http.StatusBadRequest, pesan)
		return
	}
	if (in.KelasID != k.KelasID || in.PelajaranID != k.PelajaranID) && !guruPunyaKelas(guruID, in.KelasID, in.PelajaranID) {
		guruPortalGagal(c, http.StatusForbidden, "Kelas ini belum ada di Kelas Saya")
		return
	}
	config.DB.Model(&models.Kuis{}).Where("id = ?", k.ID).Updates(map[string]interface{}{
		"judul": in.Judul, "jenis": in.Jenis, "kelas_id": in.KelasID,
		"pelajaran_id": in.PelajaranID, "deadline": selesai,
	})
	guruSimpanMetaAsesmen(k.ID, mulai, selesai)
	config.DB.Preload("Kelas").Preload("Pelajaran").First(&k, k.ID)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": guruAsesmenJSON(k)})
}

func HapusGuruAsesmen(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	k, ok := guruKuisMilik(c, guruID)
	if !ok {
		return
	}
	config.DB.Where("kuis_id = ?", k.ID).Delete(&models.SiswaAsesmenSesi{})
	config.DB.Where("kuis_id = ?", k.ID).Delete(&models.PengumpulanKuis{})
	config.DB.Where("kuis_id = ?", k.ID).Delete(&models.AsesmenMetadata{})
	config.DB.Delete(&models.Kuis{}, k.ID)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Asesmen dihapus"})
}

func DetailGuruAsesmen(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	k, ok := guruKuisMilik(c, guruID)
	if !ok {
		return
	}
	data := guruAsesmenJSON(k)
	data["soal"] = guruBacaSoal(k.Soal)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func guruValidasiSoal(in *guruSoalInput) string {
	in.Teks = strings.TrimSpace(in.Teks)
	if in.Teks == "" {
		return "Teks soal wajib diisi"
	}
	if len(in.Opsi) < 2 || len(in.Opsi) > 6 {
		return "Isi minimal 2 dan maksimal 6 opsi jawaban"
	}
	opsi := make([]guruOpsi, 0, len(in.Opsi))
	for i, o := range in.Opsi {
		t := strings.TrimSpace(o.Teks)
		if t == "" {
			return "Semua opsi jawaban harus diisi"
		}
		opsi = append(opsi, guruOpsi{Kode: string(rune('A' + i)), Teks: t})
	}
	in.Kunci = strings.ToUpper(strings.TrimSpace(in.Kunci))
	adaKunci := false
	for _, o := range opsi {
		if o.Kode == in.Kunci {
			adaKunci = true
		}
	}
	if !adaKunci {
		return "Pilih kunci jawaban yang benar"
	}
	if in.Poin < 0 {
		in.Poin = 0
	}
	in.Opsi = opsi
	in.Pembahasan = strings.TrimSpace(in.Pembahasan)
	return ""
}

func TambahGuruSoal(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	k, ok := guruKuisMilik(c, guruID)
	if !ok {
		return
	}
	var in guruSoalInput
	if err := c.ShouldBindJSON(&in); err != nil {
		guruPortalGagal(c, http.StatusBadRequest, "Data tidak valid")
		return
	}
	if pesan := guruValidasiSoal(&in); pesan != "" {
		guruPortalGagal(c, http.StatusBadRequest, pesan)
		return
	}
	soal := guruBacaSoal(k.Soal)
	maks := 0
	for _, s := range soal {
		if s.ID > maks {
			maks = s.ID
		}
	}
	soal = append(soal, guruSoal{ID: maks + 1, Teks: in.Teks, Opsi: in.Opsi, Kunci: in.Kunci, Poin: in.Poin, Pembahasan: in.Pembahasan})
	if err := guruSimpanSoal(k.ID, soal); err != nil {
		guruPortalGagal(c, http.StatusInternalServerError, "Soal gagal disimpan")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": soal})
}

func UbahGuruSoal(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	k, ok := guruKuisMilik(c, guruID)
	if !ok {
		return
	}
	nomor, err := strconv.Atoi(c.Param("nomor"))
	soal := guruBacaSoal(k.Soal)
	if err != nil || nomor < 1 || nomor > len(soal) {
		guruPortalGagal(c, http.StatusNotFound, "Soal tidak ditemukan")
		return
	}
	var in guruSoalInput
	if err := c.ShouldBindJSON(&in); err != nil {
		guruPortalGagal(c, http.StatusBadRequest, "Data tidak valid")
		return
	}
	if pesan := guruValidasiSoal(&in); pesan != "" {
		guruPortalGagal(c, http.StatusBadRequest, pesan)
		return
	}
	idx := nomor - 1
	soal[idx] = guruSoal{ID: soal[idx].ID, Teks: in.Teks, Opsi: in.Opsi, Kunci: in.Kunci, Poin: in.Poin, Pembahasan: in.Pembahasan}
	if err := guruSimpanSoal(k.ID, soal); err != nil {
		guruPortalGagal(c, http.StatusInternalServerError, "Soal gagal disimpan")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": soal})
}

func HapusGuruSoal(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	k, ok := guruKuisMilik(c, guruID)
	if !ok {
		return
	}
	nomor, err := strconv.Atoi(c.Param("nomor"))
	soal := guruBacaSoal(k.Soal)
	if err != nil || nomor < 1 || nomor > len(soal) {
		guruPortalGagal(c, http.StatusNotFound, "Soal tidak ditemukan")
		return
	}
	soal = append(soal[:nomor-1], soal[nomor:]...)
	if err := guruSimpanSoal(k.ID, soal); err != nil {
		guruPortalGagal(c, http.StatusInternalServerError, "Soal gagal dihapus")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": soal})
}

func MuridGuruAsesmen(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	k, ok := guruKuisMilik(c, guruID)
	if !ok {
		return
	}
	var siswa []models.User
	config.DB.Where("role = ? AND kelas_id = ?", "siswa", k.KelasID).Order("nama ASC").Find(&siswa)
	var sesi []models.SiswaAsesmenSesi
	config.DB.Where("kuis_id = ?", k.ID).Find(&sesi)
	byS := map[uint]models.SiswaAsesmenSesi{}
	for _, s := range sesi {
		byS[s.SiswaID] = s
	}
	rows := make([]gin.H, 0, len(siswa))
	for _, s := range siswa {
		row := gin.H{
			"siswa_id": s.ID, "nama": s.Nama, "nisn": guruPortalString(s.NISN), "foto_url": s.FotoURL,
			"kelas_nama": k.Kelas.Nama, "status": "belum", "waktu_kumpul": nil, "terlambat": false, "nilai": nil,
		}
		if ss, ada := byS[s.ID]; ada {
			if ss.SudahDikirim {
				row["status"] = "selesai"
				row["nilai"] = ss.Nilai
				if ss.DikirimAt != nil {
					row["waktu_kumpul"] = *ss.DikirimAt
					row["terlambat"] = ss.DikirimAt.After(k.Deadline)
				}
			} else {
				row["status"] = "mengerjakan"
			}
		}
		rows = append(rows, row)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"asesmen": guruAsesmenJSON(k), "murid": rows}})
}

func HasilGuruAsesmen(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	k, ok := guruKuisMilik(c, guruID)
	if !ok {
		return
	}
	siswaID := guruAtoiUint(c.Param("siswa_id"))
	var siswa models.User
	if siswaID == 0 || config.DB.Where("id = ? AND role = ? AND kelas_id = ?", siswaID, "siswa", k.KelasID).First(&siswa).Error != nil {
		guruPortalGagal(c, http.StatusNotFound, "Siswa tidak ditemukan di kelas ini")
		return
	}
	var sesi models.SiswaAsesmenSesi
	config.DB.Where("kuis_id = ? AND siswa_id = ?", k.ID, siswaID).Limit(1).Find(&sesi)
	if sesi.ID == 0 {
		guruPortalGagal(c, http.StatusNotFound, "Siswa belum mengerjakan asesmen ini")
		return
	}
	jawab := guruBacaJawaban(sesi.JawabanDraft)
	soal := guruBacaSoal(k.Soal)
	benar := 0
	detail := make([]gin.H, 0, len(soal))
	for _, s := range soal {
		jw := jawab[s.ID]
		if jw == "" {
			jw = jawab[s.Nomor]
		}
		betul := jw != "" && strings.EqualFold(jw, s.Kunci)
		if betul {
			benar++
		}
		opsi := s.Opsi
		if opsi == nil {
			opsi = []guruOpsi{}
		}
		detail = append(detail, gin.H{
			"nomor": s.Nomor, "teks": s.Teks, "opsi": opsi, "kunci": s.Kunci,
			"jawaban": jw, "benar": betul, "poin": s.Poin, "pembahasan": s.Pembahasan,
		})
	}
	var waktu interface{}
	if sesi.DikirimAt != nil {
		waktu = *sesi.DikirimAt
	}
	lulus := sesi.Nilai != nil && *sesi.Nilai >= guruKKM
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"asesmen_judul": k.Judul, "kelas_nama": k.Kelas.Nama, "pelajaran_nama": k.Pelajaran.Nama,
		"siswa_id": siswa.ID, "siswa_nama": siswa.Nama, "siswa_nisn": guruPortalString(siswa.NISN), "siswa_foto": siswa.FotoURL,
		"nilai": sesi.Nilai, "nilai_maks": sesi.NilaiMaks, "lulus": lulus, "kkm": guruKKM,
		"jumlah_benar": benar, "jumlah_soal": len(soal), "waktu_kumpul": waktu, "sudah_dikirim": sesi.SudahDikirim,
		"detail": detail,
	}})
}
