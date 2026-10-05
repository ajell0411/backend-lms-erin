package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
	"e-class-backend/models"
)

func recordAktivitas(c *gin.Context, aksi, jenis, nama string) {
	aktorNama := "Admin"
	if userID, ok := getUserIDFromContext(c); ok {
		var actor models.User
		if err := config.DB.Select("nama").First(&actor, userID).Error; err == nil && actor.Nama != "" {
			aktorNama = actor.Nama
		}
	}
	if err := config.DB.Create(&models.Aktivitas{
		AktorNama: aktorNama, Aksi: aksi, ObjekJenis: jenis, ObjekNama: nama,
	}).Error; err != nil {
		log.Printf("gagal mencatat aktivitas: %v", err)
	}
}

func aktivitasJenisRole(role string) string {
	switch role {
	case "siswa":
		return "murid"
	case "admin_kurikulum":
		return "admin kurikulum"
	case "kepala_sekolah":
		return "kepala sekolah"
	default:
		return "admin"
	}
}

func ListAktivitas(c *gin.Context) {
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Limit harus berupa angka positif"})
			return
		}
		if parsed > 100 {
			parsed = 100
		}
		limit = parsed
	}
	var rows []models.Aktivitas
	if err := config.DB.Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat aktivitas"})
		return
	}
	data := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		data = append(data, gin.H{
			"id": row.ID, "aktor_nama": row.AktorNama, "aksi": row.Aksi,
			"objek_jenis": row.ObjekJenis, "objek_nama": row.ObjekNama,
			"created_at": row.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}