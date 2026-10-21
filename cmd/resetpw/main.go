package main

import (
    "fmt"
    "log"
    "os"

    "github.com/glebarez/sqlite"
    "golang.org/x/crypto/bcrypt"
    "gorm.io/gorm"
)

func main() {
    username := "admin"
    password := "admin123"
    if len(os.Args) > 1 {
        username = os.Args[1]
    }
    if len(os.Args) > 2 {
        password = os.Args[2]
    }

    db, err := gorm.Open(sqlite.Open("eclass.db"), &gorm.Config{})
    if err != nil {
        log.Fatal(err)
    }

    hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        log.Fatal(err)
    }

    res := db.Exec("UPDATE users SET password = ? WHERE username = ?", string(hashed), username)
    if res.Error != nil {
        log.Fatal(res.Error)
    }
    if res.RowsAffected == 0 {
        log.Fatalf("user %q tidak ditemukan", username)
    }

    // verifikasi
    var tersimpan string
    db.Raw("SELECT password FROM users WHERE username = ?", username).Scan(&tersimpan)
    if bcrypt.CompareHashAndPassword([]byte(tersimpan), []byte(password)) != nil {
        log.Fatal("verifikasi gagal")
    }
    fmt.Printf("OK: password user %q sekarang %q\n", username, password)
}
