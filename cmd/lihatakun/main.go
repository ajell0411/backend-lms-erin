package main

import (
    "fmt"
    "log"
    "os"
    "strings"

    "github.com/glebarez/sqlite"
    "gorm.io/gorm"
)

func bersih(s string) string {
    s = strings.ReplaceAll(s, "\r", " ")
    s = strings.ReplaceAll(s, "\n", " ")
    return s
}

func main() {
    db, err := gorm.Open(sqlite.Open("eclass.db"), &gorm.Config{})
    if err != nil {
        log.Fatal(err)
    }

    // Tanpa argumen: semua akun. Dengan argumen: hanya role itu (contoh: siswa)
    query := "SELECT * FROM users ORDER BY role, username"
    args := []interface{}{}
    if len(os.Args) > 1 {
        query = "SELECT * FROM users WHERE role = ? ORDER BY username"
        args = append(args, os.Args[1])
    }

    rows, err := db.Raw(query, args...).Rows()
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
        fmt.Printf("--- Akun %d ---\n", no)
        for i, k := range kolom {
            if strings.Contains(strings.ToLower(k), "pass") {
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
            fmt.Printf("%-20s %s\n", k, bersih(teks))
        }
    }
    fmt.Printf("\nTotal: %d akun\n", no)
}
