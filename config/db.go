package config

import (
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"e-class-backend/models"
)

var DB *gorm.DB

func ConnectDB() {
	db, err := gorm.Open(sqlite.Open("eclass.db"), &gorm.Config{})
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
	)
	if err != nil {
		log.Fatal("Gagal migrasi:", err)
	}

	DB = db
	log.Println("✅ Database connected & migrated")
}