# 🍿 CINEMBROT — Movie Engine, Streamer & Torrent Subtitle Downloader

<p align="center">
  <img src="public/img/cinembrot_mascot_transparent.png" alt="CINEMBROT Logo" width="160" />
</p>

<p align="center">
  <b>Platform Streaming & Download Film Legal Gratis dengan Integrasi Subtitle Indonesia Otomatis.</b>
</p>

---

## 🚀 Fitur Utama

- 🎬 **Multi-Source Movie Scraper**: Mengambil metadata & stream film dari TMDB REST API, Internet Archive, Blender Open Movies, PublicDomainMovie, dan YTS.
- 💬 **Auto-Hardsub Subtitle Indonesia**: Terintegrasi dengan Torrent Client internal dan FFmpeg untuk mengunduh torrent, mencocokkan subtitle Bahasa Indonesia, dan meng-encode langsung menjadi file video MP4 siap tonton di HP, PC, dan Smart TV.
- 👁️ **Kontrol Privasi Torrent Publik**: Pilihan untuk menyembunyikan atau menampilkan link magnet/torrent mentah ke publik melalui saklar toggle di CMS Admin.
- ⚡ **Live Auto-Reload HTML**: Template web dimuat ulang secara otomatis saat disimpan (*save*), mempercepat pengembangan UI tanpa perlu me-restart server Go.
- 🛡️ **CMS Admin Lengkap**: Dashboard statistik, manajemen katalog film, sinkronisasi scraper, log aktivitas, dan pengaturan sistem.
- 🎨 **Modern Dark/Light UI**: Didesain menggunakan Tailwind CSS, responsif, dan ramah seluler.

---

## 🛠️ Prasyarat Sistem

1. **Go** (Golang) versi 1.22 atau lebih baru.
2. **MariaDB / MySQL** (Database `cinembrot`).
3. **FFmpeg** (Wajib jika menggunakan fitur hardsub rendering otomatis).

---

## 📦 Instalasi & Menjalankan

### 1. Persiapan Database
Pastikan layanan MariaDB aktif, lalu buat dan impor skema awal:
```sql
CREATE DATABASE IF NOT EXISTS cinembrot;
```
Impor skema dan data awal bawaan:
```powershell
mysql -u root -p cinembrot < schema.sql
```

### 2. Jalankan Server
Compile dan jalankan aplikasi:
```powershell
# Build binary
go build -o cinembrot.exe .

# Jalankan server web & background scheduler
.\cinembrot.exe -serve
```
Atau jalankan di latar belakang (background mode di Windows):
```powershell
Start-Process .\cinembrot.exe -ArgumentList "-serve" -WindowStyle Hidden
```

Akses website melalui browser:
- **Halaman Utama**: [http://localhost:8080](http://localhost:8080)
- **CMS Admin**: [http://localhost:8080/admin](http://localhost:8080/admin)
  - **Username**: `admin`
  - **Password**: `cinembrot123`

---

## ⚙️ Perintah CLI Tersedia

```powershell
# Jalankan web server (port :8080) & background auto-scraper
.\cinembrot.exe -serve

# Jalankan scraping berdasarkan tahun rilis (source: tmdb / archive / yts / all)
.\cinembrot.exe -by-year 2024 -pages 1 -source yts

# Validasi & periksa kesehatan link unduhan film di database (deteksi link mati)
.\cinembrot.exe -check-links

# Download & konversi gambar film di database ke format WebP lokal (Original & Thumb)
.\cinembrot.exe -convert-images

# Jalankan 1 siklus scraping terjadwal semua rentang tahun (polite mode)
.\cinembrot.exe -auto-scrape

# Jalankan daemon background scheduler mandiri (tanpa web server)
.\cinembrot.exe -daemon

# Cari & ingest metadata film langsung dari TMDb REST API
.\cinembrot.exe -tmdb "Inception" -year 2010

# Ingest film domain publik dari Internet Archive
.\cinembrot.exe -archive -archive-limit 10

# Ingest film Creative Commons dari Blender Studio (4K)
.\cinembrot.exe -openmovies
```

> ⚠️ **PENTING — KEBIJAKAN DATABASE:**
> Seluruh operasi database (`delete`, `restore`, `edit`, ataupun migrasi struktur) membutuhkan kehati-hatian ekstra dan **WAJIB meminta izin (permission)** pengguna sebelum dieksekusi. Dilarang menghapus atau merombak database yang sedang berjalan tanpa konfirmasi eksplisit.

---

## 📁 Struktur Direktori

```text
cinembrot/
├── auth/               # Modul autentikasi CMS Admin (HMAC-SHA256 & Session Cookie)
├── config/             # Konfigurasi aplikasi, env, database & API keys
├── database/           # Koneksi MariaDB, skema migration, & seed settings
├── imageprocessor/     # Konversi gambar poster/backdrop ke format WebP responsif
├── model/              # Definisi model data GORM (Movie, DownloadLink, TorrentTask, dll.)
├── pipeline/           # Pipeline agregasi & pengayaan metadata film lintas provider
├── provider/           # Adapter client penyedia data (TMDb, OMDb, Archive, Blender, YTS)
│   ├── archive/        # Client API Internet Archive
│   ├── omdb/           # Client API OMDb (Rating IMDb)
│   ├── openmovies/     # Client scraping film Creative Commons Blender Studio
│   ├── tmdb/           # Client REST API The Movie Database (TMDb)
│   └── yts/            # Client REST API YTS (YIFY Torrents)
├── public/             # Aset statis (WebP uploads, banner, maskot logo, favicon, downloads)
├── scheduler/          # Background worker otomatis & cron scraping periodik
├── scraper/            # Engine scraping Colly, HTML cleaner, & repository query
├── server/             # HTTP Web Server, middleware admin, dan route handlers
│   └── views/          # Template HTML dinamis dengan live auto-reload (F5)
├── torrentmgr/         # Download manager torrent internal & pipeline rendering hardsub FFmpeg
├── validator/          # Engine validasi kesehatan & deteksi dead link unduhan
├── schema.sql          # SQL dump database MariaDB lengkap (DDL + seed default)
└── CHANGELOG.md        # Catatan riwayat pembaruan & rilis
```

---

## 📄 Lisensi
Didistribusikan untuk tujuan edukasi dan pemutaran film domain publik / creative commons legal.

