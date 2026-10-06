package main

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"e-class-backend/config"
	"e-class-backend/models"
)

func setupTestEnvironment(t *testing.T) (http.Handler, string, uint) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	t.Setenv("JWT_SECRET", "test-secret-key-12345")
	t.Setenv("DATABASE_PATH", filepath.Join(t.TempDir(), "eclass-siswa-test.db"))

	config.ConnectDB()
	seedAdmin()
	seedDemoAsesmen()
	router := setupRouter()

	// Buat Jurusan, Kelas, dan Siswa
	jurusan := models.Jurusan{Nama: "Teknik Komputer & Jaringan", Kode: "TKJ", KepalaJurusan: "Pak Budi"}
	config.DB.Create(&jurusan)

	kelas := models.Kelas{Nama: "XII-TKJ-1", Tingkat: "XII", JurusanID: jurusan.ID}
	config.DB.Create(&kelas)

	pelajaran := models.Pelajaran{Nama: "Pemrograman Web", Kode: "PW"}
	config.DB.Create(&pelajaran)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("siswa123"), bcrypt.DefaultCost)
	username := "siswa_test"
	nisn := "1234567890"
	siswa := models.User{
		Nama:     "Siswa Tester",
		Username: &username,
		Email:    "siswa.test@example.com",
		Password: string(hashed),
		Role:     "siswa",
		NISN:     &nisn,
		Telepon:  "08123456789",
		KelasID:  &kelas.ID,
	}
	config.DB.Create(&siswa)

	// Buat Guru, Materi, Tugas, dan Kuis
	guru := models.User{Nama: "Guru Tester", Email: "guru@example.com", Role: "guru"}
	config.DB.Create(&guru)

	materi := models.Materi{
		Judul:       "Belajar HTML Lanjutan",
		Deskripsi:   "Panduan sintaks HTML5",
		FileUrl:     "/uploads/materi/html.pdf",
		PelajaranID: pelajaran.ID,
		KelasID:     kelas.ID,
		GuruID:      guru.ID,
	}
	config.DB.Create(&materi)

	tugas := models.Tugas{
		Judul:       "Tugas Membuat Form Login",
		Deskripsi:   "Buat form login HTML + CSS",
		LampiranUrl: "/uploads/tugas/petunjuk.pdf",
		Deadline:    time.Now().Add(24 * time.Hour),
		PelajaranID: pelajaran.ID,
		KelasID:     kelas.ID,
		GuruID:      guru.ID,
	}
	config.DB.Create(&tugas)

	kuis := models.Kuis{
		Judul:       "Kuis Dasar HTML",
		Jenis:       "kuis",
		PelajaranID: pelajaran.ID,
		KelasID:     kelas.ID,
		GuruID:      guru.ID,
		Deadline:    time.Now().Add(24 * time.Hour),
		Soal: `[
			{"id": 1, "nomor": 1, "teks": "Kepanjangan HTML?", "opsi": [{"kode":"A","teks":"HyperText Markup Language"},{"kode":"B","teks":"None"}], "kunci": "A"},
			{"id": 2, "nomor": 2, "teks": "Tag heading terbesar?", "opsi": [{"kode":"A","teks":"<h6>"},{"kode":"B","teks":"<h1>"}], "kunci": "B"}
		]`,
	}
	config.DB.Create(&kuis)

	// Login untuk dapatkan token
	loginResp := apiRequest(t, router, http.MethodPost, "/auth/login", map[string]any{
		"username": "siswa_test",
		"password": "siswa123",
	}, "")
	assertStatus(t, loginResp, http.StatusOK)
	token := decodeObject(t, loginResp)["token"].(string)

	return router, token, siswa.ID
}

func TestSiswaPortalEndpoints(t *testing.T) {
	router, token, _ := setupTestEnvironment(t)
	if sqlDB, err := config.DB.DB(); err == nil {
		defer sqlDB.Close()
	}

	// 1. Profil
	t.Run("GET /api/siswa/profil", func(t *testing.T) {
		resp := apiRequest(t, router, http.MethodGet, "/siswa/profil", nil, token)
		assertStatus(t, resp, http.StatusOK)
		data := decodeObject(t, resp)
		if data["success"] != true {
			t.Fatalf("expected success true, got %v", data)
		}
	})

	t.Run("PUT /api/siswa/profil", func(t *testing.T) {
		resp := apiRequest(t, router, http.MethodPut, "/siswa/profil", map[string]any{
			"no_hp": "08987654321",
		}, token)
		assertStatus(t, resp, http.StatusOK)
		profile := decodeObject(t, resp)["data"].(map[string]any)
		if profile["no_hp"] != "08987654321" {
			t.Fatalf("expected updated phone number, got %v", profile["no_hp"])
		}
	})

	// 2. Pelajaran
	t.Run("GET /api/siswa/pelajaran", func(t *testing.T) {
		resp := apiRequest(t, router, http.MethodGet, "/siswa/pelajaran", nil, token)
		assertStatus(t, resp, http.StatusOK)
		data := decodeObject(t, resp)["data"].([]any)
		if len(data) == 0 {
			t.Fatal("expected non-empty pelajaran list")
		}
	})

	// 3. Materi & Unduh
	var materiID int
	t.Run("GET /api/siswa/materi", func(t *testing.T) {
		resp := apiRequest(t, router, http.MethodGet, "/siswa/materi", nil, token)
		assertStatus(t, resp, http.StatusOK)
		items := decodeObject(t, resp)["data"].([]any)
		if len(items) == 0 {
			t.Fatal("expected at least 1 materi")
		}
		materiID = int(items[0].(map[string]any)["id"].(float64))
	})

	t.Run("GET /api/siswa/materi/:id/unduh", func(t *testing.T) {
		path := "/siswa/materi/" + jsonNumber(materiID) + "/unduh"
		req := httptest.NewRequest(http.MethodGet, "/api"+path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assertStatus(t, w, http.StatusOK)
		if w.Header().Get("Content-Disposition") == "" {
			t.Fatal("expected Content-Disposition header")
		}
	})

	// 4. Tugas & Kumpul & Unduh Lampiran
	var tugasID int
	t.Run("GET /api/siswa/tugas", func(t *testing.T) {
		resp := apiRequest(t, router, http.MethodGet, "/siswa/tugas", nil, token)
		assertStatus(t, resp, http.StatusOK)
		items := decodeObject(t, resp)["data"].([]any)
		if len(items) == 0 {
			t.Fatal("expected at least 1 tugas")
		}
		tugasID = int(items[0].(map[string]any)["id"].(float64))
	})

	t.Run("GET /api/siswa/tugas/:id", func(t *testing.T) {
		resp := apiRequest(t, router, http.MethodGet, "/siswa/tugas/"+jsonNumber(tugasID), nil, token)
		assertStatus(t, resp, http.StatusOK)
		detail := decodeObject(t, resp)["data"].(map[string]any)
		if detail["judul"] == "" {
			t.Fatal("expected valid task title")
		}
	})

	t.Run("GET /api/siswa/tugas/:id/lampiran", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/siswa/tugas/"+jsonNumber(tugasID)+"/lampiran", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assertStatus(t, w, http.StatusOK)
	})

	t.Run("POST /api/siswa/tugas/:id/kumpul", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("file", "jawaban.pdf")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.WriteString(part, "PDF dummy content jawaban tugas")
		_ = writer.WriteField("catatan", "Ini jawaban tugas saya")
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/siswa/tugas/"+jsonNumber(tugasID)+"/kumpul", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assertStatus(t, w, http.StatusOK)
		respData := decodeObject(t, w)
		if respData["success"] != true {
			t.Fatalf("expected success true, got %v", respData)
		}
	})

	// 5. Asesmen Flow
	var kuisID int
	t.Run("GET /api/siswa/asesmen", func(t *testing.T) {
		resp := apiRequest(t, router, http.MethodGet, "/siswa/asesmen?tipe=kuis", nil, token)
		assertStatus(t, resp, http.StatusOK)
		items := decodeObject(t, resp)["data"].([]any)
		if len(items) == 0 {
			t.Fatal("expected at least 1 kuis")
		}
		kuisID = int(items[0].(map[string]any)["id"].(float64))
	})

	t.Run("GET /api/siswa/asesmen/:id (tanpa kunci jawaban)", func(t *testing.T) {
		resp := apiRequest(t, router, http.MethodGet, "/siswa/asesmen/"+jsonNumber(kuisID), nil, token)
		assertStatus(t, resp, http.StatusOK)
		detail := decodeObject(t, resp)["data"].(map[string]any)
		soalList := detail["soal"].([]any)
		if len(soalList) == 0 {
			t.Fatal("expected soal list in detail")
		}
		// Pastikan TIDAK ADA kunci jawaban di respons
		for _, q := range soalList {
			qMap := q.(map[string]any)
			if _, exists := qMap["kunci"]; exists {
				t.Fatal("kunci jawaban must not be exposed to siswa!")
			}
			if _, exists := qMap["kunci_jawaban"]; exists {
				t.Fatal("kunci_jawaban must not be exposed to siswa!")
			}
		}
	})

	t.Run("POST /api/siswa/asesmen/:id/mulai (idempoten)", func(t *testing.T) {
		resp1 := apiRequest(t, router, http.MethodPost, "/siswa/asesmen/"+jsonNumber(kuisID)+"/mulai", nil, token)
		assertStatus(t, resp1, http.StatusOK)
		data1 := decodeObject(t, resp1)["data"].(map[string]any)
		waktu1 := data1["waktu_selesai"].(string)

		// Panggilan kedua harus idempoten
		resp2 := apiRequest(t, router, http.MethodPost, "/siswa/asesmen/"+jsonNumber(kuisID)+"/mulai", nil, token)
		assertStatus(t, resp2, http.StatusOK)
		data2 := decodeObject(t, resp2)["data"].(map[string]any)
		waktu2 := data2["waktu_selesai"].(string)

		if waktu1 != waktu2 {
			t.Fatalf("expected idempotent waktu_selesai, got %s vs %s", waktu1, waktu2)
		}
	})

	t.Run("PUT /api/siswa/asesmen/:id/jawaban", func(t *testing.T) {
		resp := apiRequest(t, router, http.MethodPut, "/siswa/asesmen/"+jsonNumber(kuisID)+"/jawaban", map[string]any{
			"jawaban": map[string]string{"1": "A", "2": "B"},
		}, token)
		assertStatus(t, resp, http.StatusOK)
	})

	t.Run("POST /api/siswa/asesmen/:id/kirim & 409 pada kirim ulang", func(t *testing.T) {
		resp := apiRequest(t, router, http.MethodPost, "/siswa/asesmen/"+jsonNumber(kuisID)+"/kirim", map[string]any{
			"jawaban": map[string]string{"1": "A", "2": "B"},
		}, token)
		assertStatus(t, resp, http.StatusOK)
		data := decodeObject(t, resp)["data"].(map[string]any)
		if int(data["nilai"].(float64)) != 100 {
			t.Fatalf("expected score 100, got %v", data["nilai"])
		}

		// Kirim ulang harus 409 Conflict
		respDup := apiRequest(t, router, http.MethodPost, "/siswa/asesmen/"+jsonNumber(kuisID)+"/kirim", map[string]any{
			"jawaban": map[string]string{"1": "A", "2": "B"},
		}, token)
		assertStatus(t, respDup, http.StatusConflict)
	})

	// 6. Pengumuman & Dibaca
	t.Run("GET /api/siswa/pengumuman", func(t *testing.T) {
		resp := apiRequest(t, router, http.MethodGet, "/siswa/pengumuman", nil, token)
		assertStatus(t, resp, http.StatusOK)
		data := decodeObject(t, resp)["data"].(map[string]any)
		items := data["items"].([]any)
		if len(items) == 0 {
			t.Fatal("expected announcements feed to have items")
		}
	})

	t.Run("POST /api/siswa/pengumuman/dibaca & PUT /api/siswa/pengumuman/dibaca", func(t *testing.T) {
		respPost := apiRequest(t, router, http.MethodPost, "/siswa/pengumuman/dibaca", nil, token)
		assertStatus(t, respPost, http.StatusOK)

		respPut := apiRequest(t, router, http.MethodPut, "/siswa/pengumuman/dibaca", nil, token)
		assertStatus(t, respPut, http.StatusOK)
	})

	// 7. Dashboard
	t.Run("GET /api/siswa/dashboard", func(t *testing.T) {
		resp := apiRequest(t, router, http.MethodGet, "/siswa/dashboard", nil, token)
		assertStatus(t, resp, http.StatusOK)
		data := decodeObject(t, resp)["data"].(map[string]any)
		if _, ok := data["asesmen"]; !ok {
			t.Fatal("dashboard missing asesmen field")
		}
		if _, ok := data["tugas"]; !ok {
			t.Fatal("dashboard missing tugas field")
		}
		if _, ok := data["materi"]; !ok {
			t.Fatal("dashboard missing materi field")
		}
	})

	// 8. Upload Foto Profil
	t.Run("POST /api/siswa/profil/foto", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("foto", "avatar.png")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.WriteString(part, "fake png content")
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/siswa/profil/foto", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assertStatus(t, w, http.StatusOK)
	})
}
