package config

import (
	"log"
	"os"

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

	DB = db
	log.Println("✅ Database connected & migrated")
}
