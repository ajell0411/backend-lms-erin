package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
	"e-class-backend/models"
)

func GetSiswaDashboard(c *gin.Context) {
	siswa, ok := getSiswaFromContext(c)
	if !ok {
		return
	}

	// 1. Asesmen terbaru maks 3
	var kuisList []models.Kuis
	config.DB.Preload("Pelajaran").Where("kelas_id = ?", *siswa.KelasID).
		Order("created_at DESC").Limit(3).Find(&kuisList)

	asesmenItems := make([]AsesmenSiswaItem, 0, len(kuisList))
	for _, k := range kuisList {
		info, _, _ := getAsesmenInfo(k, siswa.ID)
		asesmenItems = append(asesmenItems, info)
	}

	// 2. Tugas terbaru maks 3
	var tugasList []models.Tugas
	config.DB.Preload("Pelajaran").Preload("Guru").Preload("Kelas").
		Where("kelas_id = ?", *siswa.KelasID).
		Order("created_at DESC").Limit(3).Find(&tugasList)

	tugasItems := make([]TugasSiswaItem, 0, len(tugasList))
	for _, t := range tugasList {
		tugasItems = append(tugasItems, buildTugasSiswaItem(t, siswa.ID))
	}

	// 3. Materi terbaru maks 3
	var materis []models.Materi
	config.DB.Preload("Pelajaran").Where("kelas_id = ?", *siswa.KelasID).
		Order("created_at DESC").Limit(3).Find(&materis)

	materiItems := make([]MateriSiswaItem, 0, len(materis))
	for _, m := range materis {
		materiItems = append(materiItems, formatMateriItem(m))
	}

	sendSiswaSuccess(c, http.StatusOK, "Berhasil memuat dashboard siswa", gin.H{
		"asesmen": asesmenItems,
		"tugas":   tugasItems,
		"materi":  materiItems,
	})
}
