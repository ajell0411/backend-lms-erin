package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/glebarez/sqlite"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var dependentTables = []string{
	"pengumpulan_kuis",
	"pengumpulan_tugas",
	"materis",
	"tugas",
	"kuis",
	"pelajaran_guru",
	"pelajarans",
	"kelas",
	"jurusans",
	"aktivitas",
}

func main() {
	_ = godotenv.Load()
	preserveID := flag.Uint("preserve-admin-id", 0, "ID akun admin yang dipertahankan")
	execute := flag.Bool("execute", false, "Jalankan penghapusan setelah konfirmasi")
	confirmation := flag.String("confirm", "", "Harus sama dengan HAPUS DATA SELAIN ADMIN")
	flag.Parse()
	if *preserveID == 0 {
		log.Fatal("Wajib mengisi -preserve-admin-id dengan ID admin yang akan dipertahankan.")
	}
	path := os.Getenv("DATABASE_PATH")
	if path == "" {
		path = "eclass.db"
	}
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		log.Fatalf("Gagal membuka database: %v", err)
	}
	var preserved struct {
		ID   uint
		Role string
	}
	if err := db.Table("users").Select("id, role").Where("id = ?", *preserveID).Take(&preserved).Error; err != nil || preserved.Role != "admin" {
		log.Fatalf("ID %d bukan akun admin yang dapat dipertahankan.", *preserveID)
	}
	fmt.Printf("Database: %s\nAkun admin yang dipertahankan: ID %d\n", path, preserved.ID)
	for _, table := range dependentTables {
		var count int64
		if err := db.Table(table).Count(&count).Error; err != nil {
			log.Fatalf("Gagal menghitung tabel %s: %v", table, err)
		}
		fmt.Printf("Akan menghapus %d baris dari %s\n", count, table)
	}
	var otherUsers int64
	if err := db.Table("users").Where("id <> ?", *preserveID).Count(&otherUsers).Error; err != nil {
		log.Fatalf("Gagal menghitung akun lain: %v", err)
	}
	fmt.Printf("Akan menghapus %d akun selain ID %d\n", otherUsers, *preserveID)
	if !*execute {
		fmt.Println("Mode pratinjau: database tidak diubah. Tambahkan -execute dan -confirm \"HAPUS DATA SELAIN ADMIN\" untuk menjalankan.")
		return
	}
	if *confirmation != "HAPUS DATA SELAIN ADMIN" {
		log.Fatal("Konfirmasi tidak cocok. Tidak ada data yang dihapus.")
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		for _, table := range dependentTables {
			if err := tx.Exec("DELETE FROM " + table).Error; err != nil {
				return fmt.Errorf("gagal menghapus %s: %w", table, err)
			}
		}
		return tx.Exec("DELETE FROM users WHERE id <> ?", *preserveID).Error
	})
	if err != nil {
		log.Fatalf("Reset dibatalkan dan transaksi di-rollback: %v", err)
	}
	fmt.Println("Reset selesai. Akun admin yang dipilih tetap dipertahankan.")
}
