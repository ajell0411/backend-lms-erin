package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"

    "e-class-backend/config"
    "e-class-backend/models"
)

// GetSiswaProfilLengkap mengirim semua data siswa (read-only) tanpa username/password.
// Tidak mewajibkan siswa punya kelas, jadi tidak lagi 403 untuk siswa tanpa kelas.
func GetSiswaProfilLengkap(c *gin.Context) {
    userID, ok := getUserIDFromContext(c)
    if !ok {
        sendSiswaError(c, http.StatusUnauthorized, "Sesi Anda tidak valid atau telah berakhir")
        return
    }
    if c.GetString("role") != "siswa" {
        sendSiswaError(c, http.StatusForbidden, "Akses hanya diperbolehkan untuk akun siswa")
        return
    }

    var user models.User
    if err := config.DB.Preload("Kelas.Jurusan").First(&user, userID).Error; err != nil {
        sendSiswaError(c, http.StatusNotFound, "Data siswa tidak ditemukan")
        return
    }

    namaKelas, namaJurusan := "", ""
    if user.Kelas != nil {
        namaKelas = user.Kelas.Nama
        namaJurusan = user.Kelas.Jurusan.Nama
    }
    nisn := ""
    if user.NISN != nil {
        nisn = *user.NISN
    }
    tanggalLahir := ""
    if user.TanggalLahir != nil {
        tanggalLahir = user.TanggalLahir.Format("2006-01-02")
    }

    sendSiswaSuccess(c, http.StatusOK, "Berhasil memuat profil siswa", gin.H{
        "id":            user.ID,
        "nama":          user.Nama,
        "nisn":          nisn,
        "jurusan":       namaJurusan,
        "nama_jurusan":  namaJurusan,
        "kelas":         namaKelas,
        "nama_kelas":    namaKelas,
        "email":         user.Email,
        "no_hp":         user.Telepon,
        "telepon":       user.Telepon,
        "foto_url":      user.FotoURL,
        "jenis_kelamin": user.JenisKelamin,
        "tempat_lahir":  user.TempatLahir,
        "tanggal_lahir": tanggalLahir,
        "alamat":        user.Alamat,
        "nama_wali":     user.NamaWali,
        "telepon_wali":  user.TeleponWali,
        "tahun_masuk":   user.TahunMasuk,
        "status":        user.Status,
    })
}