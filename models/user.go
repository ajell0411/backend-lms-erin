package models

import "time"

type User struct {
	ID           uint        `gorm:"primaryKey" json:"id"`
	Nama         string      `json:"nama"`
	Username     *string     `gorm:"uniqueIndex" json:"username"`
	Email        string      `gorm:"unique" json:"email"`
	Password     string      `json:"-"`
	Role         string      `json:"role"` // admin | admin_kurikulum | kepala_sekolah | guru | siswa
	NIP          string      `json:"nip"`
	NISN         *string     `gorm:"uniqueIndex" json:"nisn"`
	JenisKelamin string      `json:"jenis_kelamin"`
	TempatLahir  string      `json:"tempat_lahir"`
	TanggalLahir *time.Time  `json:"tanggal_lahir"`
	Alamat       string      `json:"alamat"`
	Telepon      string      `json:"telepon"`
	NamaWali     string      `json:"nama_wali"`
	TeleponWali  string      `json:"telepon_wali"`
	TahunMasuk   *int        `json:"tahun_masuk"`
	Status       string      `gorm:"not null;default:aktif" json:"status"`
	FotoURL      string      `json:"foto_url"`
	KelasID      *uint       `json:"kelas_id,omitempty"`
	Kelas        *Kelas      `json:"kelas,omitempty"`
	Pelajaran    []Pelajaran `gorm:"many2many:pelajaran_guru;" json:"-"`
	CreatedAt    time.Time   `gorm:"autoCreateTime" json:"createdAt"`
}

type Jurusan struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Nama string `json:"nama"`
	Kode string `json:"kode"`
}

type Kelas struct {
	ID          uint    `gorm:"primaryKey" json:"id"`
	Nama        string  `json:"nama"`
	Tingkat     string  `json:"tingkat"`
	JurusanID   uint    `gorm:"not null" json:"jurusanId"`
	Jurusan     Jurusan `json:"jurusan"`
	WaliKelasID *uint   `json:"waliKelasId,omitempty"`
	WaliKelas   *User   `json:"waliKelas,omitempty"`
}

type Pelajaran struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Nama string `json:"nama"`
	Kode string `json:"kode"`
	Guru []User `gorm:"many2many:pelajaran_guru;" json:"guru"`
}

type Materi struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Judul       string    `json:"judul"`
	Deskripsi   string    `json:"deskripsi"`
	FileUrl     string    `json:"fileUrl"`
	PelajaranID uint      `gorm:"not null" json:"pelajaranId"`
	Pelajaran   Pelajaran `json:"pelajaran,omitempty"`
	KelasID     uint      `gorm:"not null" json:"kelasId"`
	Kelas       Kelas     `json:"kelas,omitempty"`
	GuruID      uint      `gorm:"not null" json:"guruId"`
	Guru        User      `json:"guru,omitempty"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

type Tugas struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Judul       string    `json:"judul"`
	Deskripsi   string    `json:"deskripsi"`
	LampiranUrl string    `json:"lampiranUrl"`
	Deadline    time.Time `json:"deadline"`
	PelajaranID uint      `gorm:"not null" json:"pelajaranId"`
	Pelajaran   Pelajaran `json:"pelajaran,omitempty"`
	KelasID     uint      `gorm:"not null" json:"kelasId"`
	Kelas       Kelas     `json:"kelas,omitempty"`
	GuruID      uint      `gorm:"not null" json:"guruId"`
	Guru        User      `json:"guru,omitempty"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

type PengumpulanTugas struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	TugasID         uint      `gorm:"not null;uniqueIndex:idx_tugas_siswa" json:"tugasId"`
	SiswaID         uint      `gorm:"not null;uniqueIndex:idx_tugas_siswa" json:"siswaId"`
	FileJawabanUrl  string    `json:"fileJawabanUrl"`
	Nilai           *int      `json:"nilai,omitempty"`
	Feedback        string    `json:"feedback"`
	StatusPenilaian string    `json:"statusPenilaian"`
	WaktuSubmit     time.Time `gorm:"autoCreateTime" json:"waktuSubmit"`
	Terlambat       bool      `json:"terlambat"`
}

type Kuis struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Judul       string    `json:"judul"`
	Jenis       string    `json:"jenis"` // kuis | ujian_harian
	GuruID      uint      `gorm:"not null" json:"guruId"`
	Guru        User      `json:"guru,omitempty"`
	PelajaranID uint      `gorm:"not null" json:"pelajaranId"`
	Pelajaran   Pelajaran `json:"pelajaran,omitempty"`
	KelasID     uint      `gorm:"not null" json:"kelasId"`
	Kelas       Kelas     `json:"kelas,omitempty"`
	Soal        string    `gorm:"type:text" json:"soal"`
	Deadline    time.Time `json:"deadline"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

type PengumpulanKuis struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	KuisID       uint      `gorm:"not null;uniqueIndex:idx_kuis_siswa" json:"kuisId"`
	SiswaID      uint      `gorm:"not null;uniqueIndex:idx_kuis_siswa" json:"siswaId"`
	Jawaban      string    `gorm:"type:text" json:"jawaban"`
	Nilai        *int      `json:"nilai,omitempty"`
	WaktuMulai   time.Time `json:"waktuMulai"`
	WaktuSelesai time.Time `json:"waktuSelesai"`
}
