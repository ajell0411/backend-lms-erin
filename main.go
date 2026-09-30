package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"

	"e-class-backend/config"
	"e-class-backend/handlers"
	"e-class-backend/middleware"
	"e-class-backend/models"
)

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

	admin := models.User{
		Nama:     "Super Admin",
		Email:    "admin",
		Password: string(hashed),
		Role:     "admin",
	}
	if err := config.DB.Create(&admin).Error; err != nil {
		log.Println("gagal membuat admin awal:", err)
	}
}

func main() {
	godotenv.Load()
	config.ConnectDB()
	seedAdmin()

	// Kelompok role yang dipakai berulang
	pantau := []string{"admin", "admin_kurikulum", "kepala_sekolah"}
	bacaMaster := []string{"admin", "admin_kurikulum", "kepala_sekolah", "guru"}   // tanpa siswa
	bacaAkademik := []string{"admin_kurikulum", "kepala_sekolah", "guru", "siswa"} // tanpa admin

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
		api.POST("/upload/foto", middleware.VerifyToken(), handlers.UploadFoto)

		// ===== Manajemen akun: khusus admin =====
		admin := api.Group("/admin")
		admin.Use(middleware.VerifyToken())
		{
			admin.POST("", middleware.RequireRole("admin"), handlers.CreateAdmin)
			admin.GET("", middleware.RequireRole(pantau...), handlers.ListAdmin)
			admin.GET("/:id", middleware.RequireRole(pantau...), handlers.GetAdminByID)
			admin.PUT("/:id", middleware.RequireRole("admin"), handlers.UpdateAdmin)
			admin.DELETE("/:id", middleware.RequireRole("admin"), handlers.DeleteAdmin)
		}

		kurikulum := api.Group("/admin-kurikulum")
		kurikulum.Use(middleware.VerifyToken())
		{
			kurikulum.POST("", middleware.RequireRole("admin"), handlers.CreateAkun("admin_kurikulum"))
			kurikulum.GET("", middleware.RequireRole(pantau...), handlers.ListAkun("admin_kurikulum"))
			kurikulum.GET("/:id", middleware.RequireRole(pantau...), handlers.GetAkunByID("admin_kurikulum"))
			kurikulum.PUT("/:id", middleware.RequireRole("admin"), handlers.UpdateAkun("admin_kurikulum"))
			kurikulum.DELETE("/:id", middleware.RequireRole("admin"), handlers.DeleteAkun("admin_kurikulum"))
		}

		kepsek := api.Group("/kepala-sekolah")
		kepsek.Use(middleware.VerifyToken())
		{
			kepsek.POST("", middleware.RequireRole("admin"), handlers.CreateAkun("kepala_sekolah"))
			kepsek.GET("", middleware.RequireRole(pantau...), handlers.ListAkun("kepala_sekolah"))
			kepsek.GET("/:id", middleware.RequireRole(pantau...), handlers.GetAkunByID("kepala_sekolah"))
			kepsek.PUT("/:id", middleware.RequireRole("admin"), handlers.UpdateAkun("kepala_sekolah"))
			kepsek.DELETE("/:id", middleware.RequireRole("admin"), handlers.DeleteAkun("kepala_sekolah"))
		}

		// Guru dan siswa: admin kelola, role pemantau boleh baca
		guru := api.Group("/guru")
		guru.Use(middleware.VerifyToken())
		{
			guru.POST("", middleware.RequireRole("admin"), handlers.CreateGuru)
			guru.GET("", middleware.RequireRole(pantau...), handlers.ListGuru)
			guru.GET("/:id", middleware.RequireRole(pantau...), handlers.GetGuruByID)
			guru.PUT("/:id", middleware.RequireRole("admin"), handlers.UpdateGuru)
			guru.DELETE("/:id", middleware.RequireRole("admin"), handlers.DeleteGuru)
		}

		siswa := api.Group("/siswa")
		siswa.Use(middleware.VerifyToken())
		{
			siswa.POST("", middleware.RequireRole("admin"), handlers.CreateSiswa)
			siswa.GET("", middleware.RequireRole(pantau...), handlers.ListSiswa)
			siswa.GET("/:id", middleware.RequireRole(pantau...), handlers.GetSiswaByID)
			siswa.PUT("/:id", middleware.RequireRole("admin"), handlers.UpdateSiswa)
			siswa.DELETE("/:id", middleware.RequireRole("admin"), handlers.DeleteSiswa)
		}

		// ===== Data master: admin kelola, siswa tidak boleh melihat =====
		jurusan := api.Group("/jurusan")
		jurusan.Use(middleware.VerifyToken())
		{
			jurusan.POST("", middleware.RequireRole("admin"), handlers.CreateJurusan)
			jurusan.GET("", middleware.RequireRole(bacaMaster...), handlers.ListJurusan)
			jurusan.GET("/:id", middleware.RequireRole(bacaMaster...), handlers.GetJurusanByID)
			jurusan.PUT("/:id", middleware.RequireRole("admin"), handlers.UpdateJurusan)
			jurusan.DELETE("/:id", middleware.RequireRole("admin"), handlers.DeleteJurusan)
		}

		kelas := api.Group("/kelas")
		kelas.Use(middleware.VerifyToken())
		{
			kelas.POST("", middleware.RequireRole("admin"), handlers.CreateKelas)
			kelas.GET("", middleware.RequireRole(bacaMaster...), handlers.ListKelas)
			kelas.GET("/:id", middleware.RequireRole(bacaMaster...), handlers.GetKelasByID)
			kelas.PUT("/:id", middleware.RequireRole("admin"), handlers.UpdateKelas)
			kelas.DELETE("/:id", middleware.RequireRole("admin"), handlers.DeleteKelas)
		}

		pelajaran := api.Group("/pelajaran")
		pelajaran.Use(middleware.VerifyToken())
		{
			pelajaran.POST("", middleware.RequireRole("admin"), handlers.CreatePelajaran)
			pelajaran.GET("", middleware.RequireRole(bacaMaster...), handlers.ListPelajaran)
			pelajaran.GET("/:id", middleware.RequireRole(bacaMaster...), handlers.GetPelajaranByID)
			pelajaran.PUT("/:id", middleware.RequireRole("admin"), handlers.UpdatePelajaran)
			pelajaran.DELETE("/:id", middleware.RequireRole("admin"), handlers.DeletePelajaran)
		}

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

	r.Run(":8080")
}
