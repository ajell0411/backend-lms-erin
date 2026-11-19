package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"e-class-backend/config"
	"e-class-backend/models"
)

var resetPasswordRoles = map[string]bool{
	"admin": true, "super_admin": true, "admin_kurikulum": true,
	"kepala_sekolah": true, "guru": true, "siswa": true,
}

func ListAccountsForPasswordReset(c *gin.Context) {
	query := config.DB.Where("role IN ?", []string{"admin", "super_admin", "admin_kurikulum", "kepala_sekolah", "guru", "siswa"})
	if c.GetString("role") != "super_admin" {
		query = query.Where("role <> ?", "super_admin")
	}
	var users []models.User
	if err := query.Order("nama ASC").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat daftar akun"})
		return
	}
	data := make([]gin.H, 0, len(users))
	for _, user := range users {
		data = append(data, gin.H{
			"id": user.ID, "nama": user.Nama, "username": user.Username,
			"email": user.Email, "role": user.Role,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func ResetAccountPassword(c *gin.Context) {
	role := strings.TrimSpace(c.Param("role"))
	if !resetPasswordRoles[role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role akun tidak valid"})
		return
	}
	if role == "super_admin" && c.GetString("role") != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hanya super admin yang dapat mereset kata sandi akun super admin"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID akun tidak valid"})
		return
	}
	var input struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || !validPassword(input.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kata sandi minimal 8 karakter dan hanya boleh berisi huruf dan angka"})
		return
	}
	var user models.User
	if err := config.DB.Where("id = ? AND role = ?", id, role).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Akun tidak ditemukan"})
		return
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses kata sandi"})
		return
	}
	if err := config.DB.Model(&user).Update("password", string(hashed)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kata sandi gagal diperbarui"})
		return
	}
	recordAktivitas(c, "mengubah", aktivitasJenisRole(role), "kata sandi "+user.Nama)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Kata sandi berhasil direset"})
}

type adminGuruKelasInput struct {
	GuruID      uint `json:"guru_id"`
	KelasID     uint `json:"kelas_id"`
	PelajaranID uint `json:"pelajaran_id"`
}

func adminGuruKelasJSON(item models.GuruKelas) gin.H {
	return gin.H{
		"id": item.ID, "guru_id": item.GuruID, "guru_nama": item.Guru.Nama,
		"kelas_id": item.KelasID, "kelas_nama": item.Kelas.Nama, "tingkat": item.Kelas.Tingkat,
		"pelajaran_id": item.PelajaranID, "pelajaran_nama": item.Pelajaran.Nama,
	}
}

func ListAdminGuruKelas(c *gin.Context) {
	query := config.DB.Preload("Guru").Preload("Kelas").Preload("Pelajaran")
	if raw := strings.TrimSpace(c.Query("kelas_id")); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 32)
		if err != nil || id == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID kelas tidak valid"})
			return
		}
		query = query.Where("kelas_id = ?", id)
	}
	var rows []models.GuruKelas
	if err := query.Order("id ASC").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat penugasan guru"})
		return
	}
	data := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		data = append(data, adminGuruKelasJSON(row))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func TambahAdminGuruKelas(c *gin.Context) {
	var input adminGuruKelasInput
	if err := c.ShouldBindJSON(&input); err != nil || input.GuruID == 0 || input.KelasID == 0 || input.PelajaranID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pilih guru, kelas, dan mata pelajaran"})
		return
	}
	var guru models.User
	if err := config.DB.Where("id = ? AND role = ?", input.GuruID, "guru").First(&guru).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Guru tidak ditemukan"})
		return
	}
	var kelas models.Kelas
	if err := config.DB.First(&kelas, input.KelasID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kelas tidak ditemukan"})
		return
	}
	var pelajaran models.Pelajaran
	if err := config.DB.First(&pelajaran, input.PelajaranID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mata pelajaran tidak ditemukan"})
		return
	}
	item := models.GuruKelas{GuruID: input.GuruID, KelasID: input.KelasID, PelajaranID: input.PelajaranID}
	if err := config.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Penugasan guru, kelas, dan mata pelajaran tersebut sudah ada"})
		return
	}
	if err := config.DB.Preload("Guru").Preload("Kelas").Preload("Pelajaran").First(&item, item.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Penugasan tersimpan tetapi gagal dimuat kembali"})
		return
	}
	recordAktivitas(c, "menambahkan", "kelas", guru.Nama+" mengajar "+kelas.Nama+" - "+pelajaran.Nama)
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": adminGuruKelasJSON(item)})
}

func HapusAdminGuruKelas(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID penugasan tidak valid"})
		return
	}
	var item models.GuruKelas
	if err := config.DB.Preload("Guru").First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Penugasan tidak ditemukan"})
		return
	}
	var scheduleCount int64
	if err := config.DB.Model(&models.Jadwal{}).
		Where("guru_id = ? AND kelas_id = ? AND pelajaran_id = ?", item.GuruID, item.KelasID, item.PelajaranID).
		Count(&scheduleCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Jadwal penugasan gagal diperiksa"})
		return
	}
	if scheduleCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Hapus jadwal mengajar terkait sebelum melepas penugasan guru"})
		return
	}
	if err := config.DB.Delete(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Penugasan gagal dihapus"})
		return
	}
	recordAktivitas(c, "menghapus", "kelas", item.Guru.Nama+" dari penugasan kelas")
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Penugasan guru dihapus"})
}

type adminJadwalInput struct {
	GuruKelasID uint   `json:"guru_kelas_id"`
	Hari        string `json:"hari"`
	JamMulai    string `json:"jam_mulai"`
	JamSelesai  string `json:"jam_selesai"`
	Ruangan     string `json:"ruangan"`
}

func adminJadwalJSON(item models.Jadwal) gin.H {
	var assignment models.GuruKelas
	config.DB.Where("guru_id = ? AND kelas_id = ? AND pelajaran_id = ?", item.GuruID, item.KelasID, item.PelajaranID).Limit(1).Find(&assignment)
	return gin.H{
		"id": item.ID, "guru_kelas_id": assignment.ID, "guru_id": item.GuruID,
		"guru_nama": item.Guru.Nama, "kelas_id": item.KelasID, "kelas_nama": item.Kelas.Nama,
		"pelajaran_id": item.PelajaranID, "pelajaran_nama": item.Pelajaran.Nama,
		"hari": item.Hari, "jam_mulai": item.JamMulai, "jam_selesai": item.JamSelesai,
		"ruangan": item.Ruangan,
	}
}

func adminJadwalValid(input *adminJadwalInput) (models.GuruKelas, string) {
	input.Hari = strings.ToLower(strings.TrimSpace(input.Hari))
	input.Ruangan = strings.TrimSpace(input.Ruangan)
	validDay := false
	for _, day := range guruPortalHari {
		if day == input.Hari {
			validDay = true
			break
		}
	}
	if !validDay {
		return models.GuruKelas{}, "Hari tidak valid"
	}
	start, startErr := time.Parse("15:04", input.JamMulai)
	end, endErr := time.Parse("15:04", input.JamSelesai)
	if input.GuruKelasID == 0 || startErr != nil || endErr != nil || !end.After(start) {
		return models.GuruKelas{}, "Pilih penugasan guru dan pastikan jam selesai setelah jam mulai"
	}
	var assignment models.GuruKelas
	if err := config.DB.First(&assignment, input.GuruKelasID).Error; err != nil {
		return models.GuruKelas{}, "Penugasan guru tidak ditemukan"
	}
	return assignment, ""
}

func ListAdminJadwal(c *gin.Context) {
	var rows []models.Jadwal
	if err := config.DB.Preload("Guru").Preload("Kelas").Preload("Pelajaran").Order("hari ASC").Order("jam_mulai ASC").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat jadwal"})
		return
	}
	data := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		data = append(data, adminJadwalJSON(row))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func TambahAdminJadwal(c *gin.Context) {
	var input adminJadwalInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data jadwal tidak valid"})
		return
	}
	assignment, message := adminJadwalValid(&input)
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	item := models.Jadwal{
		GuruID: assignment.GuruID, KelasID: assignment.KelasID, PelajaranID: assignment.PelajaranID,
		Hari: input.Hari, JamMulai: input.JamMulai, JamSelesai: input.JamSelesai, Ruangan: input.Ruangan,
	}
	if err := config.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Jadwal gagal disimpan"})
		return
	}
	if err := config.DB.Preload("Guru").Preload("Kelas").Preload("Pelajaran").First(&item, item.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Jadwal tersimpan tetapi gagal dimuat kembali"})
		return
	}
	recordAktivitas(c, "menambahkan", "jadwal", item.Kelas.Nama+" - "+item.Pelajaran.Nama)
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": adminJadwalJSON(item)})
}

func UbahAdminJadwal(c *gin.Context) {
	var item models.Jadwal
	if err := config.DB.First(&item, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Jadwal tidak ditemukan"})
		return
	}
	var input adminJadwalInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data jadwal tidak valid"})
		return
	}
	assignment, message := adminJadwalValid(&input)
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	updates := map[string]interface{}{
		"guru_id": assignment.GuruID, "kelas_id": assignment.KelasID, "pelajaran_id": assignment.PelajaranID,
		"hari": input.Hari, "jam_mulai": input.JamMulai, "jam_selesai": input.JamSelesai, "ruangan": input.Ruangan,
	}
	if err := config.DB.Model(&item).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Jadwal gagal diperbarui"})
		return
	}
	if err := config.DB.Preload("Guru").Preload("Kelas").Preload("Pelajaran").First(&item, item.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Jadwal diperbarui tetapi gagal dimuat kembali"})
		return
	}
	recordAktivitas(c, "mengubah", "jadwal", item.Kelas.Nama+" - "+item.Pelajaran.Nama)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": adminJadwalJSON(item)})
}

func HapusAdminJadwal(c *gin.Context) {
	var item models.Jadwal
	if err := config.DB.Preload("Kelas").Preload("Pelajaran").First(&item, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Jadwal tidak ditemukan"})
		return
	}
	if err := config.DB.Delete(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Jadwal gagal dihapus"})
		return
	}
	recordAktivitas(c, "menghapus", "jadwal", item.Kelas.Nama+" - "+item.Pelajaran.Nama)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Jadwal dihapus"})
}
