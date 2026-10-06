BAGIAN 4 dari 4: backend Go dan verifikasi akhir (tanpa gambar)

Lanjutkan SISI SISWA E-CLASS (bagian 4 dari 4): BACKEND di e-class-backend + verifikasi akhir.

ATURAN KERAS: hanya TAMBAH. Jangan ubah handler, route, middleware, atau format respons yang sudah ada untuk admin/guru. Jangan ubah cmd/reset-demo-data atau test lama. Buat branch "fitur-siswa". Cek dulu main.go, handlers/, models/user.go dan pakai ulang endpoint yang sudah ada.

Buat handlers/siswa_*.go dan daftarkan route baru di main.go di bawah /api dengan middleware JWT + pembatas role siswa. Data hanya untuk kelas/jurusan siswa yang login (ambil dari token, jangan percaya id dari client). Respons {success, message, data}, pesan error bahasa Indonesia, status HTTP benar (400/401/403/404/409/422).

Endpoint yang dibutuhkan (hanya yang belum ada):
- GET /api/siswa/dashboard: {asesmen, tugas, materi} masing-masing maks 3 terbaru
- GET /api/siswa/pengumuman?jenis=&page=&limit=: feed gabungan materi/tugas/asesmen (limit opsional, dipakai popup lonceng)
- POST /api/siswa/pengumuman/dibaca: tandai pengumuman sudah dibaca (untuk titik biru dan titik merah lonceng)
- GET /api/siswa/pelajaran: untuk dropdown
- GET /api/siswa/materi?search=&pelajaran_id= ; GET /api/siswa/materi/:id/unduh (stream file + Content-Disposition nama asli)
- GET /api/siswa/asesmen?search=&pelajaran_id=&tipe=ujian_harian|kuis (status aktif/selesai/terkunci + nilai)
- GET /api/siswa/asesmen/:id (detail + soal TANPA kunci jawaban); POST /api/siswa/asesmen/:id/mulai (catat waktu mulai, hitung waktu_selesai, idempoten); POST /api/siswa/asesmen/:id/kirim (simpan jawaban, hitung nilai pilihan ganda; sudah dikirim = 409; di luar jadwal = 403)
- GET /api/siswa/tugas?search=&pelajaran_id= ; GET /api/siswa/tugas/:id ; POST /api/siswa/tugas/:id/kumpul (multipart file + catatan; hanya .zip/.rar/.pdf, maks 10MB, nama file unik; unique per (TugasID, SiswaID), kirim ulang = update sebelum tenggat; setelah tenggat tandai terlambat)
- GET /api/siswa/profil ; PUT /api/siswa/profil ; POST /api/siswa/profil/foto (JPG/PNG maks 2MB)

Model: pakai GORM dan model yang ada (Materi, Tugas, PengumpulanTugas, Kuis, PengumpulanKuis, User, Kelas, Jurusan, Pelajaran). Field baru (mis. tipe kuis ujian_harian/kuis, durasi_menit, jadwal_mulai, jadwal_selesai, bobot) hanya ditambah lewat auto-migrate bila belum ada, nullable atau ber-default, tanpa mengubah field lama.

Hubungkan fungsi di lib/siswa.ts frontend dengan endpoint ini (ganti semua TODO), dengan menyesuaikan di sisi siswa saja.

VERIFIKASI AKHIR:
- go build ./... dan npm run build lolos.
- Tes alur: login siswa1/siswa123 lalu dashboard, popup lonceng, pengumuman, unduh materi, filter asesmen, kerjakan & kirim asesmen, kumpul tugas, edit profil, logout.
- Login admin: pastikan tampilan dan fungsi admin sama seperti sebelumnya.
- git diff --stat: kembalikan (git checkout -- file) bila ada file admin yang berubah.
- Ringkasan akhir: file dibuat/diubah, file yang diubah di luar app/siswa dan lib/siswa.ts beserta alasannya, endpoint baru, asumsi, dan bug admin yang ditemukan tapi TIDAK diperbaiki.