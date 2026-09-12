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
# Jalankan web server & cron scheduler
.\cinembrot.exe -serve

# Jalankan proses scraping manual
.\cinembrot.exe -scrape -source yts -year 2024 -pages 1

# Cek kesehatan link download & stream di database
.\cinembrot.exe -check-links

# Jalankan pengujian hardsub torrent
.\cinembrot.exe -hardsub-test
```

---

## 📁 Struktur Direktori

```text
cinebrot/
├── config/             # Konfigurasi aplikasi & database
├── database/           # Koneksi MariaDB, schema migration & seed data
├── model/              # Definisi model data GORM (Movie, DownloadLink, dll.)
├── scraper/            # Modul scraper (TMDB, Archive, Blender, YTS)
├── scheduler/          # Background worker otomatis untuk scraping berkala
├── server/             # HTTP Web Server & Route Handlers
│   └── views/          # Template HTML (Layout, detail, admin CMS)
├── torrentmgr/         # Download manager torrent & pipeline FFmpeg Hardsub
├── public/             # Aset statis (Logo, CSS, JS, Favicon, Poster)
├── schema.sql          # SQL dump database lengkap
└── CHANGELOG.md        # Catatan riwayat pembaruan
```

---

## 📄 Lisensi
Didistribusikan untuk tujuan edukasi dan pemutaran film domain publik / creative commons legal.
