package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"

	"e-class-backend/config"
	"e-class-backend/handlers"
	"e-class-backend/middleware"
	"e-class-backend/models"
)

func autoMigrateSiswaModels() {
	if config.DB != nil {
		_ = config.DB.AutoMigrate(
			&models.PengumumanDibaca{},
			&models.SiswaPengumumanStatus{},
			&models.SiswaAsesmenSesi{},
			&models.SiswaPengumpulanCatatan{},
			&models.AsesmenMetadata{},
			&models.TugasMetadata{},
			&models.GuruKelas{},
			&models.Jadwal{},
		)
	}
}

func seedDemoAsesmen() {
	if config.DB == nil {
		return
	}
	var count int64
	config.DB.Model(&models.Kuis{}).Count(&count)
	if count > 0 {
		return
	}

	var guru models.User
	if err := config.DB.Where("role = ?", "guru").First(&guru).Error; err != nil {
		return
	}

	var kelas models.Kelas
	if err := config.DB.First(&kelas).Error; err != nil {
		return
	}

	var pelajaran models.Pelajaran
	if err := config.DB.First(&pelajaran).Error; err != nil {
		return
	}

	soalKuis := `[
		{"id": 1, "nomor": 1, "teks": "Apa kepanjangan dari HTML?", "opsi": [
			{"kode": "A", "teks": "Hyper Text Markup Language"},
			{"kode": "B", "teks": "High Tech Multi Language"},
			{"kode": "C", "teks": "Hyper Transfer Markup Language"},
			{"kode": "D", "teks": "Home Tool Markup Language"}
		], "kunci": "A"},
		{"id": 2, "nomor": 2, "teks": "Tag HTML manakah yang digunakan untuk membuat heading terbesar?", "opsi": [
			{"kode": "A", "teks": "<heading>"},
			{"kode": "B", "teks": "<h6>"},
			{"kode": "C", "teks": "<h1>"},
			{"kode": "D", "teks": "<head>"}
		], "kunci": "C"},
		{"id": 3, "nomor": 3, "teks": "Properti CSS apa yang digunakan untuk mengatur warna latar belakang?", "opsi": [
			{"kode": "A", "teks": "color"},
			{"kode": "B", "teks": "background-color"},
			{"kode": "C", "teks": "bgcolor"},
			{"kode": "D", "teks": "background-image"}
		], "kunci": "B"}
	]`

	soalUjian := `[
		{"id": 1, "nomor": 1, "teks": "Manakah atribut HTML yang digunakan untuk menentukan link tujuan pada tag <a>?", "opsi": [
			{"kode": "A", "teks": "src"},
			{"kode": "B", "teks": "href"},
			{"kode": "C", "teks": "link"},
			{"kode": "D", "teks": "target"}
		], "kunci": "B"},
		{"id": 2, "nomor": 2, "teks": "Simbol CSS yang digunakan untuk memilih elemen berdasarkan ID adalah?", "opsi": [
			{"kode": "A", "teks": "."},
			{"kode": "B", "teks": "#"},
			{"kode": "C", "teks": "@"},
			{"kode": "D", "teks": "*"}
		], "kunci": "B"}
	]`

	now := time.Now()
	kuis1 := models.Kuis{
		Judul:       "Kuis HTML & CSS Dasar",
		Jenis:       "kuis",
		GuruID:      guru.ID,
		PelajaranID: pelajaran.ID,
		KelasID:     kelas.ID,
		Soal:        soalKuis,
		Deadline:    now.Add(14 * 24 * time.Hour),
	}
	config.DB.Create(&kuis1)

	kuis2 := models.Kuis{
		Judul:       "Ujian Harian Pemrograman Web",
		Jenis:       "ujian_harian",
		GuruID:      guru.ID,
		PelajaranID: pelajaran.ID,
		KelasID:     kelas.ID,
		Soal:        soalUjian,
		Deadline:    now.Add(14 * 24 * time.Hour),
	}
	config.DB.Create(&kuis2)
}

func seedAdmin() {
	var count int64
	config.DB.Model(&models.User{}).Where("role = ?", "admin").Count(&count)
	if count > 0 {
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Println("gagal hash password admin:", err)
		return
	}

	username := "admin"
	admin := models.User{
		Nama:     "Admin E-CLASS",
		Username: &username,
		Email:    "admin",
		Password: string(hashed),
		Role:     "admin",
	}
	if err := config.DB.Create(&admin).Error; err != nil {
		log.Println("gagal membuat admin awal:", err)
	}
}

func seedSuperAdmin() {
	var count int64
	config.DB.Model(&models.User{}).Where("role = ?", "super_admin").Count(&count)
	if count > 0 {
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte("superadmin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Println("gagal hash password super admin:", err)
		return
	}

	username := "superadmin"
	superAdmin := models.User{
		Nama:     "Super Admin E-CLASS",
		Username: &username,
		Email:    "superadmin",
		Password: string(hashed),
		Role:     "super_admin",
	}
	if err := config.DB.Create(&superAdmin).Error; err != nil {
		log.Println("gagal membuat super admin awal:", err)
	}
}

func setupRouter() *gin.Engine {
	autoMigrateSiswaModels()

	// Kelompok role yang dipakai berulang
	pantau := []string{"admin", "super_admin", "admin_kurikulum", "kepala_sekolah"}
	bacaMaster := []string{"admin", "super_admin", "admin_kurikulum", "kepala_sekolah", "guru"} // tanpa siswa
	bacaAkademik := []string{"admin_kurikulum", "kepala_sekolah", "guru", "siswa"}              // tanpa admin

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	r := gin.Default()
	r.SetTrustedProxies(nil)

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{frontendURL},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Authorization", "Content-Type"},
	}))
	r.Static("/uploads", "./uploads")

	r.GET("/api/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Backend Go jalan!"})
	})

	api := r.Group("/api")
	{
		// ===== Auth =====
		api.POST("/auth/login", handlers.Login)
		profile := api.Group("/profile")
		profile.Use(middleware.VerifyToken())
		{
			profile.GET("", handlers.GetProfile)
			profile.PUT("", handlers.UpdateProfile)
			profile.PUT("/password", handlers.UpdateProfilePassword)
		}
		api.POST("/upload/foto", middleware.VerifyToken(), handlers.UploadFoto)

		// ===== Manajemen akun: khusus admin =====
		admin := api.Group("/admin")
		admin.Use(middleware.VerifyToken())
		{
			admin.POST("", middleware.RequireRole("super_admin"), handlers.CreateAdmin)
			admin.GET("", middleware.RequireRole(pantau...), handlers.ListAdmin)
			admin.GET("/:id", middleware.RequireRole(pantau...), handlers.GetAdminByID)
			admin.PUT("/:id", middleware.RequireRole("super_admin"), handlers.UpdateAdmin)
			admin.DELETE("/:id", middleware.RequireRole("super_admin"), handlers.DeleteAdmin)
		}

		kurikulum := api.Group("/admin-kurikulum")
		kurikulum.Use(middleware.VerifyToken())
		{
			kurikulum.POST("", middleware.RequireAdminWrite(), handlers.CreateAkun("admin_kurikulum"))
			kurikulum.GET("", middleware.RequireRole(pantau...), handlers.ListAkun("admin_kurikulum"))
			kurikulum.GET("/:id", middleware.RequireRole(pantau...), handlers.GetAkunByID("admin_kurikulum"))
			kurikulum.PUT("/:id", middleware.RequireAdminWrite(), handlers.UpdateAkun("admin_kurikulum"))
			kurikulum.DELETE("/:id", middleware.RequireAdminWrite(), handlers.DeleteAkun("admin_kurikulum"))
		}

		kepsek := api.Group("/kepala-sekolah")
		kepsek.Use(middleware.VerifyToken())
		{
			kepsek.POST("", middleware.RequireAdminWrite(), handlers.CreateAkun("kepala_sekolah"))
			kepsek.GET("", middleware.RequireRole(pantau...), handlers.ListAkun("kepala_sekolah"))
			kepsek.GET("/:id", middleware.RequireRole(pantau...), handlers.GetAkunByID("kepala_sekolah"))
			kepsek.PUT("/:id", middleware.RequireAdminWrite(), handlers.UpdateAkun("kepala_sekolah"))
			kepsek.DELETE("/:id", middleware.RequireAdminWrite(), handlers.DeleteAkun("kepala_sekolah"))
		}

		// Guru dan siswa: admin kelola, role pemantau boleh baca
		guru := api.Group("/guru")
		guru.Use(middleware.VerifyToken())
		{
			guru.POST("", middleware.RequireAdminWrite(), handlers.CreateGuru)
			guru.GET("", middleware.RequireRole(pantau...), handlers.ListGuru)
			guru.GET("/:id", middleware.RequireRole(pantau...), handlers.GetGuruByID)
			guru.PUT("/:id", middleware.RequireAdminWrite(), handlers.UpdateGuru)
			guru.DELETE("/:id", middleware.RequireAdminWrite(), handlers.DeleteGuru)

			// ===== Fitur Guru (Role Guru) =====
			guru.GET("/dashboard", middleware.RequireRole("guru"), handlers.GetGuruDashboard)
			guru.GET("/kelas", middleware.RequireRole("guru"), handlers.ListGuruKelas)
			guru.POST("/kelas", middleware.RequireRole("guru"), handlers.TambahGuruKelas)
			guru.GET("/kelas/:id", middleware.RequireRole("guru"), handlers.DetailGuruKelas)
			guru.PUT("/kelas/:id", middleware.RequireRole("guru"), handlers.UbahGuruKelas)
			guru.DELETE("/kelas/:id", middleware.RequireRole("guru"), handlers.HapusGuruKelas)
			guru.GET("/jadwal", middleware.RequireRole("guru"), handlers.ListGuruJadwal)
			guru.POST("/jadwal", middleware.RequireRole("guru"), handlers.TambahGuruJadwal)
			guru.PUT("/jadwal/:id", middleware.RequireRole("guru"), handlers.UbahGuruJadwal)
			guru.DELETE("/jadwal/:id", middleware.RequireRole("guru"), handlers.HapusGuruJadwal)
			guru.GET("/materi", middleware.RequireRole("guru"), handlers.ListGuruMateri)
			guru.POST("/materi", middleware.RequireRole("guru"), handlers.TambahGuruMateri)
			guru.PUT("/materi/:id", middleware.RequireRole("guru"), handlers.UbahGuruMateri)
			guru.DELETE("/materi/:id", middleware.RequireRole("guru"), handlers.HapusGuruMateri)
			guru.GET("/materi/:id/unduh", middleware.RequireRole("guru"), handlers.UnduhGuruMateri)
			guru.GET("/tugas", middleware.RequireRole("guru"), handlers.ListGuruTugas)
			guru.POST("/tugas", middleware.RequireRole("guru"), handlers.TambahGuruTugas)
			guru.GET("/tugas/:id", middleware.RequireRole("guru"), handlers.DetailGuruTugas)
			guru.PUT("/tugas/:id", middleware.RequireRole("guru"), handlers.UbahGuruTugas)
			guru.DELETE("/tugas/:id", middleware.RequireRole("guru"), handlers.HapusGuruTugas)
			guru.POST("/tugas/:id/nonaktif", middleware.RequireRole("guru"), handlers.NonaktifkanGuruTugas)
			guru.GET("/pengumpulan/:id", middleware.RequireRole("guru"), handlers.DetailGuruPengumpulan)
			guru.GET("/pengumpulan/:id/file", middleware.RequireRole("guru"), handlers.FileGuruPengumpulan)
			guru.PUT("/pengumpulan/:id/nilai", middleware.RequireRole("guru"), handlers.NilaiGuruPengumpulan)
			guru.GET("/asesmen", middleware.RequireRole("guru"), handlers.ListGuruAsesmen)
			guru.POST("/asesmen", middleware.RequireRole("guru"), handlers.TambahGuruAsesmen)
			guru.GET("/asesmen/:id", middleware.RequireRole("guru"), handlers.DetailGuruAsesmen)
			guru.PUT("/asesmen/:id", middleware.RequireRole("guru"), handlers.UbahGuruAsesmen)
			guru.DELETE("/asesmen/:id", middleware.RequireRole("guru"), handlers.HapusGuruAsesmen)
			guru.POST("/asesmen/:id/soal", middleware.RequireRole("guru"), handlers.TambahGuruSoal)
			guru.PUT("/asesmen/:id/soal/:nomor", middleware.RequireRole("guru"), handlers.UbahGuruSoal)
			guru.DELETE("/asesmen/:id/soal/:nomor", middleware.RequireRole("guru"), handlers.HapusGuruSoal)
			guru.GET("/asesmen/:id/murid", middleware.RequireRole("guru"), handlers.MuridGuruAsesmen)
			guru.GET("/asesmen/:id/hasil/:siswa_id", middleware.RequireRole("guru"), handlers.HasilGuruAsesmen)
			guru.GET("/nilai", middleware.RequireRole("guru"), handlers.RekapGuruNilai)
			guru.GET("/aktivitas", middleware.RequireRole("guru"), handlers.ListGuruAktivitas)
		}

		siswa := api.Group("/siswa")
		siswa.Use(middleware.VerifyToken())
		{
			siswa.POST("", middleware.RequireAdminWrite(), handlers.CreateSiswa)
			siswa.GET("", middleware.RequireRole(pantau...), handlers.ListSiswa)
			siswa.GET("/:id", middleware.RequireRole(pantau...), handlers.GetSiswaByID)
			siswa.PUT("/:id", middleware.RequireAdminWrite(), handlers.UpdateSiswa)
			siswa.DELETE("/:id", middleware.RequireAdminWrite(), handlers.DeleteSiswa)

			// ===== Fitur Siswa (Role Siswa) =====
			siswa.GET("/dashboard", middleware.RequireRole("siswa"), handlers.GetSiswaDashboard)
			siswa.GET("/pengumuman", middleware.RequireRole("siswa"), handlers.ListSiswaPengumuman)
			siswa.POST("/pengumuman/dibaca", middleware.RequireRole("siswa"), handlers.TandaiSiswaPengumumanDibaca)
			siswa.PUT("/pengumuman/dibaca", middleware.RequireRole("siswa"), handlers.TandaiSiswaPengumumanDibaca)
			siswa.GET("/pelajaran", middleware.RequireRole("siswa"), handlers.ListSiswaPelajaran)
			siswa.GET("/materi", middleware.RequireRole("siswa"), handlers.ListSiswaMateri)
			siswa.GET("/materi/:id/unduh", middleware.RequireRole("siswa"), handlers.UnduhSiswaMateri)
			siswa.GET("/asesmen", middleware.RequireRole("siswa"), handlers.ListSiswaAsesmen)
			siswa.GET("/asesmen/:id", middleware.RequireRole("siswa"), handlers.GetSiswaAsesmenByID)
			siswa.POST("/asesmen/:id/mulai", middleware.RequireRole("siswa"), handlers.MulaiSiswaAsesmen)
			siswa.PUT("/asesmen/:id/jawaban", middleware.RequireRole("siswa"), handlers.SimpanJawabanSiswaAsesmen)
			siswa.POST("/asesmen/:id/kirim", middleware.RequireRole("siswa"), handlers.KirimSiswaAsesmen)
			siswa.GET("/tugas", middleware.RequireRole("siswa"), handlers.ListSiswaTugas)
			siswa.GET("/tugas/:id", middleware.RequireRole("siswa"), handlers.GetSiswaTugasByID)
			siswa.GET("/tugas/:id/lampiran", middleware.RequireRole("siswa"), handlers.UnduhSiswaTugasLampiran)
			siswa.POST("/tugas/:id/kumpul", middleware.RequireRole("siswa"), handlers.KumpulkanSiswaTugas)
			siswa.GET("/profil", middleware.RequireRole("siswa"), handlers.GetSiswaProfilLengkap)
			siswa.PUT("/profil", middleware.RequireRole("siswa"), handlers.UpdateSiswaProfil)
			siswa.POST("/profil/foto", middleware.RequireRole("siswa"), handlers.UploadFotoSiswaProfil)
		}

		// ===== Data master: admin kelola, siswa tidak boleh melihat =====
		jurusan := api.Group("/jurusan")
		jurusan.Use(middleware.VerifyToken())
		{
			jurusan.POST("", middleware.RequireAdminWrite(), handlers.CreateJurusan)
			jurusan.GET("", middleware.RequireRole(bacaMaster...), handlers.ListJurusan)
			jurusan.GET("/:id", middleware.RequireRole(bacaMaster...), handlers.GetJurusanByID)
			jurusan.PUT("/:id", middleware.RequireAdminWrite(), handlers.UpdateJurusan)
			jurusan.DELETE("/:id", middleware.RequireAdminWrite(), handlers.DeleteJurusan)
		}

		kelas := api.Group("/kelas")
		kelas.Use(middleware.VerifyToken())
		{
			kelas.POST("", middleware.RequireAdminWrite(), handlers.CreateKelas)
			kelas.GET("", middleware.RequireRole(bacaMaster...), handlers.ListKelas)
			kelas.GET("/:id", middleware.RequireRole(bacaMaster...), handlers.GetKelasByID)
			kelas.PUT("/:id", middleware.RequireAdminWrite(), handlers.UpdateKelas)
			kelas.DELETE("/:id", middleware.RequireAdminWrite(), handlers.DeleteKelas)
		}

		pelajaran := api.Group("/pelajaran")
		pelajaran.Use(middleware.VerifyToken())
		{
			pelajaran.POST("", middleware.RequireAdminWrite(), handlers.CreatePelajaran)
			pelajaran.GET("", middleware.RequireRole(bacaMaster...), handlers.ListPelajaran)
			pelajaran.GET("/:id", middleware.RequireRole(bacaMaster...), handlers.GetPelajaranByID)
			pelajaran.GET("/:id/guru", middleware.RequireRole(bacaMaster...), handlers.ListGuruPelajaran)
			pelajaran.PUT("/:id", middleware.RequireAdminWrite(), handlers.UpdatePelajaran)
			pelajaran.DELETE("/:id", middleware.RequireAdminWrite(), handlers.DeletePelajaran)
		}

		// ===== Pemantauan (read only): statistik dan aktivitas guru =====
		pemantauan := api.Group("/pemantauan")
		pemantauan.Use(middleware.VerifyToken(), middleware.RequireRole(pantau...))
		{
			pemantauan.GET("/statistik", handlers.GetPemantauanStatistik)
			pemantauan.GET("/aktivitas-guru", handlers.ListPemantauanAktivitasGuru)
			pemantauan.GET("/aktivitas-guru/:jenis/:id", handlers.GetPemantauanAktivitasGuruDetail)
		}
		api.GET("/aktivitas", middleware.VerifyToken(), middleware.RequireRole(pantau...), handlers.ListAktivitas)

		// ===== Pembelajaran: guru yang kelola, admin tidak ikut =====
		materi := api.Group("/materi")
		materi.Use(middleware.VerifyToken())
		{
			materi.POST("", middleware.RequireRole("guru"), handlers.CreateMateri)
			materi.GET("", middleware.RequireRole(bacaAkademik...), handlers.ListMateri)
			materi.GET("/:id", middleware.RequireRole(bacaAkademik...), handlers.GetMateriByID)
			materi.PUT("/:id", middleware.RequireRole("guru"), handlers.UpdateMateri)
			materi.DELETE("/:id", middleware.RequireRole("guru"), handlers.DeleteMateri)
		}

		tugas := api.Group("/tugas")
		tugas.Use(middleware.VerifyToken())
		{
			tugas.POST("", middleware.RequireRole("guru"), handlers.CreateTugas)
			tugas.GET("", middleware.RequireRole(bacaAkademik...), handlers.ListTugas)
			tugas.GET("/:id", middleware.RequireRole(bacaAkademik...), handlers.GetTugasByID)
			tugas.PUT("/:id", middleware.RequireRole("guru"), handlers.UpdateTugas)
			tugas.DELETE("/:id", middleware.RequireRole("guru"), handlers.DeleteTugas)
		}

		// ===== Pengumpulan tugas dan penilaian =====
		pengumpulan := api.Group("/pengumpulan")
		pengumpulan.Use(middleware.VerifyToken())
		{
			pengumpulan.POST("", middleware.RequireRole("siswa"), handlers.CreatePengumpulan)
			pengumpulan.GET("", middleware.RequireRole("guru", "admin_kurikulum", "kepala_sekolah"), handlers.ListPengumpulanByTugas)
			pengumpulan.GET("/saya", middleware.RequireRole("siswa"), handlers.ListPengumpulanSaya)
			pengumpulan.GET("/:id", middleware.RequireRole(bacaAkademik...), handlers.GetPengumpulanByID)
			pengumpulan.PUT("/:id/nilai", middleware.RequireRole("guru"), handlers.BeriNilaiPengumpulan)
			pengumpulan.DELETE("/:id", middleware.RequireRole("guru", "siswa"), handlers.DeletePengumpulan)
		}

		// ===== Kuis/Ujian: guru yang kelola, admin tidak ikut =====
		kuis := api.Group("/kuis")
		kuis.Use(middleware.VerifyToken())
		{
			kuis.POST("", middleware.RequireRole("guru"), handlers.CreateKuis)
			kuis.GET("", middleware.RequireRole(bacaAkademik...), handlers.ListKuis)
			kuis.GET("/:id", middleware.RequireRole(bacaAkademik...), handlers.GetKuisByID)
			kuis.PUT("/:id", middleware.RequireRole("guru"), handlers.UpdateKuis)
			kuis.DELETE("/:id", middleware.RequireRole("guru"), handlers.DeleteKuis)
		}

		// ===== Pengumpulan kuis dan penilaian =====
		pengumpulanKuis := api.Group("/pengumpulan-kuis")
		pengumpulanKuis.Use(middleware.VerifyToken())
		{
			pengumpulanKuis.POST("", middleware.RequireRole("siswa"), handlers.CreatePengumpulanKuis)
			pengumpulanKuis.GET("", middleware.RequireRole("guru", "admin_kurikulum", "kepala_sekolah"), handlers.ListPengumpulanKuisByKuis)
			pengumpulanKuis.GET("/saya", middleware.RequireRole("siswa"), handlers.ListPengumpulanKuisSaya)
			pengumpulanKuis.GET("/:id", middleware.RequireRole(bacaAkademik...), handlers.GetPengumpulanKuisByID)
			pengumpulanKuis.PUT("/:id/nilai", middleware.RequireRole("guru"), handlers.BeriNilaiKuis)
		}
	}

	return r
}

func main() {
	godotenv.Load()
	config.ConnectDB()
	seedAdmin()
	seedSuperAdmin()
	seedDemoAsesmen()

	r := setupRouter()
	handlers.BackfillGuruKelas()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Gagal menjalankan server:", err)
	}
}
