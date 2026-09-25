package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "golang.org/x/crypto/bcrypt"

    "e-class-backend/config"
    "e-class-backend/models"
)

type siswaInput struct {
    Nama     string `json:"nama"`
    Email    string `json:"email"`
    Password string `json:"password"`
    KelasID  *uint  `json:"kelasId"`
}

func CreateSiswa(c *gin.Context) {
    var input siswaInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal hash password"})
        return
    }

    user := models.User{
        Nama:     input.Nama,
        Email:    input.Email,
        Password: string(hashed),
        Role:     "siswa",
        KelasID:  input.KelasID,
    }

    if err := config.DB.Create(&user).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusCreated, user)
}

// ListSiswa mendukung filter ?kelas=<id> seperti versi Next.js lama
func ListSiswa(c *gin.Context) {
    var users []models.User
    query := config.DB.Where("role = ?", "siswa")

    kelasID := c.Query("kelas")
    if kelasID != "" {
        query = query.Where("kelas_id = ?", kelasID)
    }

    query.Find(&users)
    c.JSON(http.StatusOK, users)
}

func GetSiswaByID(c *gin.Context) {
    var user models.User
    if err := config.DB.Where("role = ?", "siswa").First(&user, c.Param("id")).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
        return
    }
    c.JSON(http.StatusOK, user)
}

func UpdateSiswa(c *gin.Context) {
    var user models.User
    if err := config.DB.Where("role = ?", "siswa").First(&user, c.Param("id")).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Tidak ditemukan"})
        return
    }

    var input siswaInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    user.Nama = input.Nama
    user.Email = input.Email
    user.KelasID = input.KelasID

    // Password cuma diupdate kalau dikirim (nggak kosong)
    if input.Password != "" {
        hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal hash password"})
            return
        }
        user.Password = string(hashed)
    }

    config.DB.Save(&user)
    c.JSON(http.StatusOK, user)
}

func DeleteSiswa(c *gin.Context) {
    if err := config.DB.Where("role = ?", "siswa").Delete(&models.User{}, c.Param("id")).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "Berhasil dihapus"})
}
