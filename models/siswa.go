package models

import "time"

type PengumumanDibaca struct {
	ID       uint      `gorm:"primaryKey" json:"id"`
	SiswaID  uint      `gorm:"not null;index" json:"siswa_id"`
	Jenis    string    `gorm:"not null" json:"jenis"`
	RefID    uint      `gorm:"not null" json:"ref_id"`
	DibacaAt time.Time `gorm:"autoCreateTime" json:"dibaca_at"`
}

type SiswaPengumumanStatus struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	SiswaID        uint      `gorm:"not null;uniqueIndex" json:"siswa_id"`
	TerakhirDibaca time.Time `json:"terakhir_dibaca"`
}

type SiswaAsesmenSesi struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	KuisID       uint       `gorm:"not null;uniqueIndex:idx_sesi_kuis_siswa" json:"kuis_id"`
	SiswaID      uint       `gorm:"not null;uniqueIndex:idx_sesi_kuis_siswa" json:"siswa_id"`
	WaktuMulai   time.Time  `json:"waktu_mulai"`
	WaktuSelesai time.Time  `json:"waktu_selesai"`
	JawabanDraft string     `gorm:"type:text" json:"jawaban_draft"`
	SudahDikirim bool       `gorm:"default:false" json:"sudah_dikirim"`
	DikirimAt    *time.Time `json:"dikirim_at"`
	Nilai        *int       `json:"nilai"`
	NilaiMaks    int        `gorm:"default:100" json:"nilai_maks"`
}

type SiswaPengumpulanCatatan struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TugasID   uint      `gorm:"not null;uniqueIndex:idx_catatan_tugas_siswa" json:"tugas_id"`
	SiswaID   uint      `gorm:"not null;uniqueIndex:idx_catatan_tugas_siswa" json:"siswa_id"`
	Catatan   string    `gorm:"type:text" json:"catatan"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type AsesmenMetadata struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	KuisID        uint       `gorm:"not null;uniqueIndex" json:"kuis_id"`
	DurasiMenit   int        `gorm:"default:60" json:"durasi_menit"`
	JadwalMulai   *time.Time `json:"jadwal_mulai"`
	JadwalSelesai *time.Time `json:"jadwal_selesai"`
	Bobot         int        `gorm:"default:100" json:"bobot"`
}

type TugasMetadata struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	TugasID   uint   `gorm:"not null;uniqueIndex" json:"tugas_id"`
	Bobot     int    `gorm:"default:10" json:"bobot"`
	Instruksi string `gorm:"type:text" json:"instruksi"`
	Kriteria  string `gorm:"type:text" json:"kriteria"`
}
