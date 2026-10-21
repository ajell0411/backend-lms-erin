package main

import (
    "bufio"
    "fmt"
    "log"
    "os"
    "strings"

    "golang.org/x/crypto/bcrypt"
    "gorm.io/gorm"

    "e-class-backend/config"
    "e-class-backend/models"
)

type akun struct {
    Nama     string
    Username string
    Password string
    Role     string
}

var akunTetap = []akun{
    {"Admin", "admin", "admin123", "admin"},
    {"Siswa 1", "siswa1", "siswa123", "siswa"},
    {"Admin Kurikulum", "kurikulum", "kurikulum123", "admin_kurikulum"},
    {"Kepala Sekolah", "kepsek", "kepsek123", "kepala_sekolah"},
    {"Guru", "guru", "guru123", "guru"},
}

func main() {
    config.ConnectDB()
    db := config.DB

    sqlDB, err := db.DB()
    if err != nil {
        log.Fatal(err)
    }
    sqlDB.SetMaxOpenConns(1)
    db.Exec("PRAGMA foreign_keys = OFF")

    var tables []string
    if err := db.Raw("SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'").Scan(&tables).Error; err != nil {
        log.Fatal(err)
    }
    var hasSeq int64
    db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE name = 'sqlite_sequence'").Scan(&hasSeq)

    fmt.Println("Isi database yang akan DIHAPUS:")
    for _, t := range tables {
        var n int64
        db.Raw("SELECT COUNT(*) FROM \"" + t + "\"").Scan(&n)
        fmt.Printf("  %-28s %d baris\n", t, n)
    }
    fmt.Println("\nLalu dibuat ulang 5 akun:")
    for _, a := range akunTetap {
        fmt.Printf("  %-10s (%s)\n", a.Username, a.Role)
    }
    fmt.Print("\nKetik HAPUS untuk melanjutkan: ")
    input, _ := bufio.NewReader(os.Stdin).ReadString('\n')
    if strings.TrimSpace(input) != "HAPUS" {
        fmt.Println("Dibatalkan, tidak ada yang diubah.")
        return
    }

    err = db.Transaction(func(tx *gorm.DB) error {
        for _, t := range tables {
            if err := tx.Exec("DELETE FROM \"" + t + "\"").Error; err != nil {
                return err
            }
        }
        if hasSeq > 0 {
            if err := tx.Exec("DELETE FROM sqlite_sequence").Error; err != nil {
                return err
            }
        }
        for _, a := range akunTetap {
            hash, err := bcrypt.GenerateFromPassword([]byte(a.Password), bcrypt.DefaultCost)
            if err != nil {
                return err
            }
            username := a.Username
            user := models.User{
                Nama:     a.Nama,
                Username: &username,
                Email:    a.Username + "@eclass.local",
                Password: string(hash),
                Role:     a.Role,
                Status:   "aktif",
            }
            if err := tx.Create(&user).Error; err != nil {
                return err
            }
        }
        return nil
    })
    if err != nil {
        log.Fatal("Gagal, tidak ada yang berubah: ", err)
    }
    fmt.Println("Selesai. Database sekarang hanya berisi 5 akun.")
}