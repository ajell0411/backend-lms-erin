package models

import "time"

// GuruKelas: penugasan guru ke kelas untuk satu pelajaran.
type GuruKelas struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	GuruID      uint      `gorm:"not null;uniqueIndex:idx_guru_kelas_pel" json:"guruId"`
	KelasID     uint      `gorm:"not null;uniqueIndex:idx_guru_kelas_pel" json:"kelasId"`
	Kelas       Kelas     `json:"kelas,omitempty"`
	PelajaranID uint      `gorm:"not null;uniqueIndex:idx_guru_kelas_pel" json:"pelajaranId"`
	Pelajaran   Pelajaran `json:"pelajaran,omitempty"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

// Jadwal: jadwal mengajar guru. Hari: senin..minggu, jam format "15:04".
type Jadwal struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	GuruID      uint      `gorm:"not null;index" json:"guruId"`
	KelasID     uint      `gorm:"not null" json:"kelasId"`
	Kelas       Kelas     `json:"kelas,omitempty"`
	PelajaranID uint      `gorm:"not null" json:"pelajaranId"`
	Pelajaran   Pelajaran `json:"pelajaran,omitempty"`
	Hari        string    `gorm:"not null" json:"hari"`
	JamMulai    string    `gorm:"not null" json:"jamMulai"`
	JamSelesai  string    `gorm:"not null" json:"jamSelesai"`
	Ruangan     string    `json:"ruangan"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`
}
