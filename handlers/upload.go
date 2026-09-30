package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

const maxPhotoSize = 2 << 20

func UploadFoto(c *gin.Context) {
	file, header, err := c.Request.FormFile("foto")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "File foto wajib diunggah"})
		return
	}
	defer file.Close()

	if header.Size > maxPhotoSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "message": "Ukuran foto maksimal 2 MB"})
		return
	}

	content, err := io.ReadAll(io.LimitReader(file, maxPhotoSize+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Gagal membaca file foto"})
		return
	}
	if len(content) == 0 || len(content) > maxPhotoSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "message": "Ukuran foto maksimal 2 MB"})
		return
	}

	contentType := http.DetectContentType(content)
	extension := ""
	switch contentType {
	case "image/jpeg":
		extension = ".jpg"
	case "image/png":
		extension = ".png"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Format foto harus JPG atau PNG"})
		return
	}

	nameBytes := make([]byte, 16)
	if _, err := rand.Read(nameBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Gagal membuat nama file"})
		return
	}
	if err := os.MkdirAll("uploads", 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Gagal menyiapkan penyimpanan foto"})
		return
	}

	filename := hex.EncodeToString(nameBytes) + extension
	destination := filepath.Join("uploads", filename)
	if err := os.WriteFile(destination, content, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Gagal menyimpan foto"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "url": "/uploads/" + filename})
}
