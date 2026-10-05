package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"e-class-backend/models"
)

var DB *gorm.DB

func ConnectDB() {
	databasePath := os.Getenv("DATABASE_PATH")
	if databasePath == "" {
		databasePath = "eclass.db"
	}
	db, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal konek database:", err)
	}

	err = db.AutoMigrate(
		&models.User{},
		&models.Jurusan{},
		&models.Kelas{},
		&models.Pelajaran{},
		&models.Materi{},
		&models.Tugas{},
		&models.PengumpulanTugas{},
		&models.Kuis{},
		&models.PengumpulanKuis{},
		&models.Aktivitas{},
	)
	if err != nil {
		log.Fatal("Gagal migrasi:", err)
	}
	if err := backfillUsernames(db); err != nil {
		log.Fatal("Gagal mengisi username akun lama:", err)
	}

	DB = db
	log.Println("✅ Database connected & migrated")
}

func backfillUsernames(db *gorm.DB) error {
	var users []models.User
	if err := db.Where("username IS NULL OR TRIM(username) = ''").Order("id").Find(&users).Error; err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, user := range users {
			base := strings.TrimSpace(strings.SplitN(user.Email, "@", 2)[0])
			base = strings.Map(func(char rune) rune {
				if char >= 'A' && char <= 'Z' {
					return char + ('a' - 'A')
				}
				if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '.' || char == '_' || char == '-' {
					return char
				}
				return -1
			}, base)
			if base == "" {
				base = fmt.Sprintf("user%d", user.ID)
			}
			username := base
			for suffix := 2; ; suffix++ {
				var count int64
				if err := tx.Model(&models.User{}).Where("LOWER(username) = LOWER(?) AND id <> ?", username, user.ID).Count(&count).Error; err != nil {
					return err
				}
				if count == 0 {
					break
				}
				username = fmt.Sprintf("%s%d", base, suffix)
			}
			if err := tx.Model(&models.User{}).Where("id = ?", user.ID).Update("username", username).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
