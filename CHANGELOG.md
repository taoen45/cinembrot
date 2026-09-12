# Changelog

Semua perubahan dan pembaruan penting pada proyek **CINEMBROT** dicatat dalam dokumen ini.

---

## [Unreleased] - 2026-09-03

### 🌟 Fitur Baru (Added)
- **Live / Auto-Reload Template HTML**:
  - Penambahan pemanggilan `s.loadTemplates()` secara dinamis di `RenderHTML` (`server/server.go`).
  - Perubahan pada file `.html` di direktori `server/views/` kini langsung aktif cukup dengan menekan tombol **F5** di browser tanpa perlu me-restart server Go.
- **Pengaturan Visibilitas Link Torrent Publik (`show_torrent_public`)**:
  - Penambahan konfigurasi baru di tabel `system_settings` dengan nilai default `false`.
  - Saklar interaktif pada dashboard CMS Admin (`/admin/tools`) untuk mengontrol apakah link torrent mentah (YTS/Magnet) ditampilkan ke pengunjung publik atau disembunyikan.
  - Halaman detail film (`detail.html`) kini hanya menampilkan blok unduhan torrent mentah jika saklar ini diaktifkan oleh admin.
- **Logo Maskot Resmi & Favicon CINEMBROT**:
  - Karakter maskot berkonsep penonton bioskop gembrot dengan kacamata 3D, popcorn, dan klaket film.
  - Background transparan murni dengan *tight bounding box cropping* (642x642) agar memenuhi kanvas dan tampil tajam.
  - Warna merah badan maskot dikalibrasi presisi ke `#E50914` (*Cinema Scarlet Red*) agar menyatu dengan palet brand CINEMBROT.
  - Dipasang sebagai favicon tab browser (`favicon.png`, `favicon.ico`), logo navbar (`h-11 w-auto`), logo sidebar admin, halaman login CMS, serta logo besar di footer website (`h-24 w-auto`).
- **Template Helpers**:
  - Menambahkan fungsi helper template `cleanQuality` dan `cleanProvider` di `server/server.go` untuk membersihkan penamaan server dan label kualitas secara otomatis.

### 🎨 Tampilan & Desain UI (Changed)
- **Redesain Kartu Unduhan Film Matang (Hardsub Indonesia)**:
  - Mengubah layout grid yang sempit menjadi *full-width flexbox* yang lapang dan lega.
  - Menghilangkan redundansi teks yang berulang (*Hardsub Indonesia* tidak lagi muncul berkali-kali).
  - Merapikan badge: Label server bersih (`Server Lokal`), badge kualitas spesifik (`720p (BLURAY)`), badge subtitle terpisah (`Hardsub Indo`), dan tombol `[ Download MP4 ]` di sisi kanan yang rapi tanpa tumpang tindih.
- **Footer Website Baru**:
  - Penambahan maskot Cinembrot berukuran besar (`h-24`) dengan efek interaktif hover dan drop-shadow.
  - Penambahan tagline resmi platform streaming legal dan navigasi cepat footer.
- **Navbar & Sidebar Layout**:
  - Memperbesar ukuran logo navbar dari 36px menjadi 44px (`h-11 w-auto`) tanpa terpotong bingkai lingkaran (*unclipped standalone silhouette*).

### 🗄️ Database & Konfigurasi (Fixed & Seeded)
- **Dump Default `scrape_sources`**:
  - Menambahkan data bawaan 6 sumber scraping (TMDB, Internet Archive, Blender Open Movies, PublicDomainMovie, YTS) ke dalam `schema.sql` dan `database/schema.sql`.
- **Kredensial Default**:
  - Sinkronisasi akun CMS Admin default: `admin` / `cinembrot123`.
- **Manajemen Torrent & Hardsub Link**:
  - Memperbarui `torrentmgr/manager.go` agar link hasil render hardsub otomatis didaftarkan dengan provider `Server Lokal` dan nama kualitas yang bersih.
