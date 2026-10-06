package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
	"e-class-backend/models"
)

type PelajaranSiswaItem struct {
	ID   uint   `json:"id"`
	Nama string `json:"nama"`
	Kode string `json:"kode,omitempty"`
}

func ListSiswaPelajaran(c *gin.Context) {
	if _, ok := getSiswaFromContext(c); !ok {
		return
	}

	var pelajarans []models.Pelajaran
	if err := config.DB.Order("nama ASC").Find(&pelajarans).Error; err != nil {
		sendSiswaError(c, http.StatusInternalServerError, "Gagal mengambil daftar pelajaran")
		return
	}

	items := make([]PelajaranSiswaItem, 0, len(pelajarans))
	for _, p := range pelajarans {
		items = append(items, PelajaranSiswaItem{
			ID:   p.ID,
			Nama: p.Nama,
			Kode: p.Kode,
		})
	}

	sendSiswaSuccess(c, http.StatusOK, "Berhasil memuat daftar pelajaran", items)
}
