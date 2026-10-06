package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"e-class-backend/config"
	"e-class-backend/models"
)

type ProfilSiswaResponse struct {
	ID          uint   `json:"id"`
	Nama        string `json:"nama"`
	Username    string `json:"username"`
	NISN        string `json:"nisn"`
	Jurusan     string `json:"jurusan"`
	NamaJurusan string `json:"nama_jurusan"`
	Email       string `json:"email"`
	NoHP        string `json:"no_hp"`
	Telepon     string `json:"telepon"`
	Kelas       string `json:"kelas"`
	NamaKelas   string `json:"nama_kelas"`
	FotoURL     string `json:"foto_url"`
}

func formatProfilResponse(u models.User) ProfilSiswaResponse {
	username := ""
	if u.Username != nil {
		username = *u.Username
	}
	nisn := ""
	if u.NISN != nil {
		nisn = *u.NISN
	}
	namaKelas := ""
	namaJurusan := ""
	if u.Kelas != nil {
		namaKelas = u.Kelas.Nama
		namaJurusan = u.Kelas.Jurusan.Nama
	}

	return ProfilSiswaResponse{
		ID:          u.ID,
		Nama:        u.Nama,
		Username:    username,
		NISN:        nisn,
		Jurusan:     namaJurusan,
		NamaJurusan: namaJurusan,
		Email:       u.Email,
		NoHP:        u.Telepon,
		Telepon:     u.Telepon,
		Kelas:       namaKelas,
		NamaKelas:   namaKelas,
		FotoURL:     u.FotoURL,
	}
}

func GetSiswaProfil(c *gin.Context) {
	siswa, ok := getSiswaFromContext(c)
	if !ok {
		return
	}

	sendSiswaSuccess(c, http.StatusOK, "Berhasil memuat profil siswa", formatProfilResponse(*siswa))
}

type updateSiswaProfilInput struct {
	NoHP         string `json:"no_hp"`
	Telepon      string `json:"telepon"`
	PasswordLama string `json:"password_lama"`
	PasswordBaru string `json:"password_baru"`
}

func UpdateSiswaProfil(c *gin.Context) {
	siswa, ok := getSiswaFromContext(c)
	if !ok {
		return
	}

	var input updateSiswaProfilInput
	if err := c.ShouldBindJSON(&input); err != nil {
		sendSiswaError(c, http.StatusBadRequest, "Data input tidak valid")
		return
	}

	noHp := strings.TrimSpace(input.NoHP)
	if noHp == "" {
		noHp = strings.TrimSpace(input.Telepon)
	}
	siswa.Telepon = noHp

	changingPassword := input.PasswordLama != "" || input.PasswordBaru != ""
	if changingPassword {
		if input.PasswordLama == "" || input.PasswordBaru == "" {
			sendSiswaError(c, http.StatusBadRequest, "Password lama dan password baru wajib diisi")
			return
		}
		if len(input.PasswordBaru) < 6 {
			sendSiswaError(c, http.StatusBadRequest, "Password baru minimal 6 karakter")
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(siswa.Password), []byte(input.PasswordLama)); err != nil {
			sendSiswaError(c, http.StatusBadRequest, "Password lama tidak sesuai")
			return
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(input.PasswordBaru), bcrypt.DefaultCost)
		if err != nil {
			sendSiswaError(c, http.StatusInternalServerError, "Gagal mengenkripsi password baru")
			return
		}
		siswa.Password = string(hashed)
	}

	if err := config.DB.Save(siswa).Error; err != nil {
		sendSiswaError(c, http.StatusInternalServerError, "Gagal memperbarui profil")
		return
	}

	sendSiswaSuccess(c, http.StatusOK, "Profil berhasil diperbarui", formatProfilResponse(*siswa))
}

func UploadFotoSiswaProfil(c *gin.Context) {
	siswa, ok := getSiswaFromContext(c)
	if !ok {
		return
	}

	fileHeader, err := c.FormFile("foto")
	if err != nil {
		fileHeader, err = c.FormFile("file")
	}
	if err != nil {
		sendSiswaError(c, http.StatusBadRequest, "File foto profil wajib diunggah")
		return
	}

	// Maksimal 2 MB
	if fileHeader.Size > 2*1024*1024 {
		sendSiswaError(c, http.StatusBadRequest, "Ukuran file foto melebihi batas maksimal 2 MB")
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		sendSiswaError(c, http.StatusBadRequest, "Format foto harus JPG atau PNG")
		return
	}

	uploadDir := "./uploads/profil"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		sendSiswaError(c, http.StatusInternalServerError, "Gagal menyiapkan direktori upload")
		return
	}

	uniqueFilename := fmt.Sprintf("profil_%d_%d%s", siswa.ID, time.Now().UnixNano(), ext)
	dst := filepath.Join(uploadDir, uniqueFilename)
	if err := c.SaveUploadedFile(fileHeader, dst); err != nil {
		sendSiswaError(c, http.StatusInternalServerError, "Gagal menyimpan foto")
		return
	}

	fotoURL := fmt.Sprintf("/uploads/profil/%s", uniqueFilename)
	siswa.FotoURL = fotoURL
	if err := config.DB.Model(&models.User{}).Where("id = ?", siswa.ID).Update("foto_url", fotoURL).Error; err != nil {
		sendSiswaError(c, http.StatusInternalServerError, "Gagal memperbarui foto profil di database")
		return
	}

	sendSiswaSuccess(c, http.StatusOK, "Foto profil berhasil diperbarui", gin.H{
		"foto_url": fotoURL,
		"url":      fotoURL,
	})
}
