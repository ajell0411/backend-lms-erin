package main

import (
    "fmt"
    "log"
    "os"
    "strings"

    "github.com/glebarez/sqlite"
    "gorm.io/gorm"
)

func potong(s string, n int) string {
    r := []rune(strings.ReplaceAll(s, "\n", " "))
    if len(r) > n {
        return string(r[:n]) + "..."
    }
    return string(r)
}

func main() {
    db, err := gorm.Open(sqlite.Open("eclass.db"), &gorm.Config{})
    if err != nil {
        log.Fatal(err)
    }

    var tabel []string
    if err := db.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name").Scan(&tabel).Error; err != nil {
        log.Fatal(err)
    }

    // Tanpa argumen: tampilkan daftar tabel + jumlah baris
    if len(os.Args) < 2 {
        fmt.Printf("%-30s %s\n", "TABEL", "JUMLAH DATA")
        for _, t := range tabel {
            var n int
            db.Raw("SELECT COUNT(*) FROM `" + t + "`").Scan(&n)
            fmt.Printf("%-30s %d\n", t, n)
        }
        fmt.Println("\nUntuk lihat isi satu tabel: go run ./cmd/lihatdata NAMA_TABEL")
        return
    }

    // Dengan argumen: tampilkan isi tabel itu
    nama := os.Args[1]
    ada := false
    for _, t := range tabel {
        if t == nama {
            ada = true
        }
    }
    if !ada {
        log.Fatalf("tabel %q tidak ada. Jalankan tanpa nama tabel untuk lihat daftar.", nama)
    }

    rows, err := db.Raw("SELECT * FROM `" + nama + "`").Rows()
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()

    kolom, _ := rows.Columns()
    no := 0
    for rows.Next() {
        vals := make([]interface{}, len(kolom))
        ptrs := make([]interface{}, len(kolom))
        for i := range vals {
            ptrs[i] = &vals[i]
        }
        if err := rows.Scan(ptrs...); err != nil {
            log.Fatal(err)
        }
        no++
        fmt.Printf("--- Data %d ---\n", no)
        for i, k := range kolom {
            if strings.Contains(strings.ToLower(k), "password") {
                continue
            }
            var teks string
            switch v := vals[i].(type) {
            case nil:
                teks = "(kosong)"
            case []byte:
                teks = string(v)
            default:
                teks = fmt.Sprint(v)
            }
            fmt.Printf("%-20s %s\n", k, potong(teks, 80))
        }
    }
    fmt.Printf("\nTotal ditampilkan: %d data\n", no)
}

