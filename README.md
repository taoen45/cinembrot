# 🍿 CINEMBROT — Movie Engine, Streamer & Torrent Subtitle Downloader

<p align="center">
  <img src="public/img/cinembrot_mascot_transparent.png" alt="CINEMBROT Logo" width="160" />
</p>

<p align="center">
  <b>Platform Streaming & Download Film, Anime, dan Drama Asia Legal Gratis dengan Integrasi Subtitle Dwibahasa (ID/EN) & Multi-Server Video Player.</b>
</p>

---

## 🚀 Fitur Utama

- 📺 **Multi-Server Streaming Player**: Dilengkapi 5 server embed streaming video instan (**Server 1: VidSrc HD**, **Server 2: AutoEmbed Fast**, **Server 3: 2Embed VIP**, **Server 4: VidLink Pro**, **Server 5: SuperEmbed Multi**, serta **Trailer Resmi YouTube**) yang dapat dipilih penonton secara fleksibel langsung di halaman detail tanpa reload halaman.
- 📑 **Episode Selector Bar**: Navigasi pemilih episode interaktif client-side untuk serial Anime dan Drama Pendek, memudahkan penonton berpindah antar episode dengan mulus.
- 🔤 **Resolusi Judul English Standar QWERTY & Alias Name**: Judul berbahasa non-Latin (Kanji Jepang, Hanzi Mandarin, Hangeul Korea) secara otomatis dikonversi ke versi **English resmi standar keyboard QWERTY** via TMDb Translations API (`iso_639_1 == "en"`) dan Jikan. Judul asli tetap dipertahankan pada kolom `original_title` dan `alternative_titles` (*alias name*), serta URL slug selalu bersih (contoh: `異世界かるてっと` $\rightarrow$ **`Isekai Quartet`** / `/movie/isekai-quartet-2019`).
- 📝 **Auto-Fill Sinopsis Kosong**: Pengecekan otomatis saat proses generate/scraping; jika sinopsis berbahasa Indonesia kosong, sistem otomatis melengkapinya dari sinopsis resmi bahasa Inggris sehingga tidak ada lagi film/anime bersinopsis kosong.
- 📅 **Scraper Berdasarkan Tahun Rilis**: Kemampuan menyaring dan mengumpulkan seluruh anime atau drama Asia yang rilis pada tahun tertentu (misal: `-year 2026`) secara akurat.
- 🚀 **Arsitektur SEO Terbaik & Fast-Indexing Mesin Pencari**:
  - **Dynamic XML Sitemap (`/sitemap.xml`)**: Otomatis menghasilkan sitemap XML standar protokol Sitemaps.org 0.9 dengan ekstensi Google Image (`xmlns:image`) untuk homepage, katalog utama, dan ribuan film dengan tag `<lastmod>`, `<changefreq>`, dan `<image:image>`.
  - **Crawler Directive (`/robots.txt`)**: Standar kontrol akses perayap bot yang bersih, ramah crawler, dan mengarahkan otomatis ke sitemap resmi.
  - **Meta Tags Lengkap**: Dynamic meta description, dynamic meta keywords kaya kata kunci pencarian, canonical URL otomatis, dan tag robots `index, follow, max-image-preview:large, max-snippet:-1`.
  - **OpenGraph & Twitter Cards**: Tampilan preview visual saat link dibagikan ke WhatsApp, Telegram, Facebook, dan Twitter/X.
  - **Schema.org Rich Snippets (JSON-LD)**: Format `Movie` dan `TVSeries` dengan `AggregateRating` agar bintang kuning rating film muncul di hasil pencarian Google SERP, serta schema `WebSite` dengan `SearchAction` (Sitelinks Searchbox) di beranda.
  - **CMS Webmaster Verifications**: Kontrol setting di `/admin/settings` untuk memasukkan kode verifikasi Google Search Console, Bing Webmaster, dan Yandex tanpa edit kode HTML.
- 🌐 **Fitur Terjemahan Dwibahasa (ID / EN) & Auto-Translate Sinopsis**:
  - Bahasa Indonesia sebagai bahasa utama (*default*) dan English sebagai bahasa kedua.
  - Dropdown pemilih bahasa elegan dengan ikon bendera SVG asli berwarna di desktop & mobile sub-navbar.
  - **Auto-Translate Sinopsis Dwibahasa**: Saat scraping, sinopsis bahasa Inggris (atau bahasa non-Latin/Kanji) otomatis diterjemahkan ke Bahasa Indonesia (`synopsis`) dan versi aslinya disimpan di `synopsis_en`.
  - Mode 🇮🇩 ID menampilkan sinopsis Bahasa Indonesia yang alami, sedangkan mode 🇬🇧 EN menampilkan sinopsis Bahasa Inggris resmi.
  - Terintegrasi engine Google Website Translator untuk terjemahan menyeluruh.
- 📥 **Scraper Link Download Langsung & Refresh Broken Link Mandiri**:
  - Link download video matang langsung disiapkan dari sumber scraper (Archive MP4, Open Movies, Torrent, dan Subtitle).
  - Tombol **"🔄 Cek / Segarkan Link"** di halaman detail film: jika link download broken, sistem secara mandiri melakukan rescrape ke website sumber untuk mencari URL download baru.
  - Jika URL download dari website sumber sama persis (tidak ada link baru), tombol otomatis **disabled** untuk menjaga efisiensi server.
- 📢 **Manajemen Iklan Adsterra & CMS Pengaturan Dinamis (`/admin/settings`)**:
  - Kontrol fleksibel saklar ON/OFF dan input script untuk 10 unit iklan: Popunder, Social Bar, Banner 728x90 Header, Native Banner Rekomendasi, Direct Smartlink, Floating Sticky Skyscraper Samping (160x600 & 160x300), Mobile Sticky Bottom 320x50, Pre-footer 468x60, dan Medium Rectangle 300x250.
  - Pengaturan branding situs (Nama Situs, Tagline, Saklar Bahasa, Saklar Komentar) langsung dari antarmuka web admin tanpa perlu restart server.
- 📱 **Desain Mobile Smartphone Matang & Ramah Sentuhan**:
  - Slider hero beranda mendukung gesture **Touch Swipe** jempol (`touchstart`, `touchend`).
  - Bebas horizontal scroll overflow di smartphone berkat penyembunyian banner lebar responsif.
  - Bantalan safe area bawah (`pb-24`) agar floating sticky ad tidak menutupi tombol konten.
- 🌟 **Katalog Khusus Film Hollywood & Box Office**:
  - Halaman khusus `/hollywood` (dan `/box-office`) bernuansa cinematic amber/emas dengan Hero Banner, filter genre lengkap, tahun rilis, rating, dan sorting cerdas.
  - Scraper resmi TMDb Movie API untuk kategori *Box Office*, *Popular*, *Top Rated*, dan *Now Playing*.
  - Pemrosesan gambar WebP lokal otomatis, link subtitle dwibahasa (ID/EN), dan auto-embed 5 player streaming.
  - Manajemen di CMS Admin (`/admin/hollywood`) dan tombol scraping instan di `/admin/tools`.
- 🎌 **Katalog Khusus Anime & Drama Pendek**:
  - Halaman khusus `/anime` dan `/drama-pendek` dengan Hero Banner, filter genre, dan status rilis.
  - Scraping anime resmi via MyAnimeList / Jikan API v4 dengan fallback TMDb.
  - Scraping drama Asia (K-Drama Korea, C-Drama China, J-Drama Jepang, Thai Drama) via TMDb TV API.
- 💬 **Auto-Subtitle Dwibahasa (Indonesia & English)**: Kandidat unduhan subtitle SRT otomatis dicari dan disiapkan dari SubDL dan OpenSubtitles.
- 🖼️ **Optimalisasi Storage WebP (Hemat 90%)**: Download dan konversi otomatis poster serta backdrop ke format WebP lokal (Full & Thumbnail) dengan *Smart Placeholder SVG* jika gambar belum terunduh.
- 🎬 **Multi-Source Legal Movie Scraper**: Agregasi metadata dari TMDb REST API, Internet Archive, Blender Open Movies, PublicDomainMovie, dan YTS.
- ⚡ **Live Auto-Reload HTML**: Template web dimuat ulang otomatis saat file template disimpan tanpa me-restart service Go.

---

## 🛠️ Prasyarat Sistem

1. **Go (Golang)** versi 1.22 atau lebih baru.
2. **MariaDB / MySQL** (Database `cinembrot`).
3. **FFmpeg** (Opsional, untuk fitur torrent hardsub rendering).

---

## 📦 Instalasi & Menjalankan

### 1. Persiapan Database
Pastikan MariaDB aktif, buat database:
```sql
CREATE DATABASE IF NOT EXISTS cinembrot;
```
Impor skema awal (jika database baru):
```powershell
mysql -u root -p cinembrot < schema.sql
```

### 2. Kompilasi & Panduan Operasional Server (Menyalakan & Mematikan)

#### A. Menyalakan Web Server
Pilih salah satu cara berikut di terminal PowerShell:

```powershell
# Opsi 1: Dijalankan langsung di Terminal (Rekomendasi saat Uji Coba / Dev)
# Anda dapat melihat log inisialisasi database dan trafik HTTP secara langsung
.\cinembrot.exe -serve
# atau via Go source code:
go run . -serve

# Opsi 2: Dijalankan di Background (Latar Belakang - Terminal tetap bebas dipakai)
Start-Process .\cinembrot.exe -ArgumentList "-serve" -WindowStyle Hidden
```
> ⏱️ **Catatan Waktu Inisialisasi**: Saat pertama kali dinyalakan, aplikasi membutuhkan waktu **sekitar 3-5 detik** untuk inisialisasi koneksi MariaDB remote, sinkronisasi skema tabel, dan pengaturan sistem sebelum port `:8080` siap menerima pengunjung.

#### B. Mematikan Web Server
```powershell
# Jika dijalankan di Terminal (Opsi 1):
Tekan kombinasi tombol keyboard: Ctrl + C

# Jika dijalankan di Background (Opsi 2) atau ingin mematikan paksa seluruh proses:
Stop-Process -Name "cinembrot" -Force
```

#### C. Memeriksa Apakah Server Sedang Berjalan
```powershell
# Cek apakah proses cinembrot aktif
Get-Process -Name "cinembrot" -ErrorAction SilentlyContinue

# Cek apakah port :8080 sedang aktif mendengarkan (LISTEN)
Get-NetTCPConnection -LocalPort 8080 -ErrorAction SilentlyContinue
```

#### D. Akses Situs Melalui Browser:
Gunakan URL protokol HTTP murni:
- **Halaman Utama**: [http://localhost:8080](http://localhost:8080)
- **Katalog Anime**: [http://localhost:8080/anime](http://localhost:8080/anime)
- **Drama Pendek / Asia**: [http://localhost:8080/drama-pendek](http://localhost:8080/drama-pendek)
- **Film Hollywood / Box Office**: [http://localhost:8080/hollywood](http://localhost:8080/hollywood)
- **CMS Admin**: [http://localhost:8080/admin](http://localhost:8080/admin)
  - **Username**: `admin`
  - **Password**: `cinembrot123`
- **Pengaturan Situs & Iklan CMS**: [http://localhost:8080/admin/settings](http://localhost:8080/admin/settings)

---

## ⚙️ Perintah CLI Terminal Lengkap (Internal Commands)

Aplikasi menyediakan berbagai opsi CLI terminal (setara *Artisan* pada Laravel) untuk mempermudah automasi, pemeliharaan, dan scraping:

### 1. Operasional Web Server & Scheduler
```powershell
# [UTAMA & WAJIB] Jalankan web server di port :8080 beserta auto-scraper background
# Menghandle seluruh routing web, CMS admin, API, dan cron otomatis di background.
.\cinembrot.exe -serve

# [OPSIONAL / WORKER MANDIRI] Jalankan daemon scheduler otomatis tanpa web server
# Berguna jika ingin memisahkan proses cron/worker di background container atau systemd terpisah.
.\cinembrot.exe -daemon

# [LEGACY SCRAPER] Jalankan 1 siklus scraping ramah server (polite mode) untuk seluruh rentang tahun
# Catatan: Untuk scraping yang jauh lebih cepat, terarah, dan otomatis cek seluruh page,
# disarankan memakai command spesifik di bagian 2 & 3 (-scrape-hollywood, -scrape-anime, -scrape-drama).
.\cinembrot.exe -auto-scrape
```

### 2. Scraper Film Hollywood & Box Office (Auto-Detect Seluruh Halaman)
```powershell
# 🚀 AUTO-ALL: Scrape SEMUA halaman film Hollywood rilis tahun 2026 (sistem otomatis cek total page)
.\cinembrot.exe -scrape-hollywood -year 2026

# 🚀 AUTO-ALL: Scrape SEMUA halaman film Box Office rilis bioskop tahun 2024
.\cinembrot.exe -scrape-hollywood -year 2024

# Scrape film Box Office terpopuler (opsional batas halaman: -hollywood-pages 1 atau 2)
.\cinembrot.exe -scrape-hollywood -hollywood-cat boxoffice -hollywood-pages 2

# Scrape film Hollywood kategori rating tertinggi (Top Rated)
.\cinembrot.exe -scrape-hollywood -hollywood-cat top_rated -hollywood-pages 1

# Scrape film Hollywood yang sedang tayang di bioskop (Now Playing)
.\cinembrot.exe -scrape-hollywood -hollywood-cat now_playing -hollywood-pages 1
```

### 3. Scraper Anime & Drama Asia (Auto-Detect Seluruh Halaman)
```powershell
# 🚀 AUTO-ALL: Scrape SEMUA anime rilis tahun 2026 (otomatis cek berapa page yang ada lalu scrape semuanya)
.\cinembrot.exe -scrape-anime -year 2026

# Scrape anime tahun 2026 dengan batasan halaman tertentu (contoh: 2 halaman = ~50 anime)
.\cinembrot.exe -scrape-anime -year 2026 -anime-pages 2

# Scrape anime terpopuler sepanjang masa (MyAnimeList / TMDb)
.\cinembrot.exe -scrape-anime -anime-cat top -anime-pages 2

# Scrape anime musim ini / on-going (Seasonal)
.\cinembrot.exe -scrape-anime -anime-cat seasonal -anime-pages 1

# 🚀 AUTO-ALL: Scrape SEMUA drama Asia rilis tahun 2026
.\cinembrot.exe -scrape-drama -year 2026

# Scrape K-Drama Korea rilis tahun 2026 (atau batasi halaman: -drama-pages 2)
.\cinembrot.exe -scrape-drama -drama-lang ko -year 2026

# Scrape Drama China (C-Drama) / Mini-series
.\cinembrot.exe -scrape-drama -drama-lang zh -drama-pages 1

# Scrape Drama Jepang (J-Drama) atau Drama Thailand
.\cinembrot.exe -scrape-drama -drama-lang ja -drama-pages 1
.\cinembrot.exe -scrape-drama -drama-lang th -drama-pages 1
```

### 4. Pemeliharaan, Perbaikan Data, Download Link & Sinkronisasi Streaming
```powershell
# 📥 Isi dan perbarui URL link download (Torrent YTS 720p/1080p/4K & Subtitle Resmi ID/EN) secara massal di DB
.\cinembrot.exe -populate-downloads

# 🌐 Sinkronisasi terjemahan sinopsis dwibahasa (terjemahkan sinopsis Inggris ke Indonesia & isi synopsis_en)
.\cinembrot.exe -translate-synopsis

# 🛠️ Perbaiki judul non-Latin (Kanji/CJK) ke English QWERTY, slug bersih & isi sinopsis kosong di DB
.\cinembrot.exe -fix-titles

# 🎬 Isi & perbarui server streaming embed (VidSrc, AutoEmbed, 2Embed, VidLink) untuk semua judul di DB
.\cinembrot.exe -populate-streams

# 🔍 Validasi & scan kesehatan tautan download film di database (deteksi link mati/rusak)
.\cinembrot.exe -check-links

# 🖼️ Download & konversi gambar poster/backdrop di database ke WebP lokal (Original & Thumb)
.\cinembrot.exe -convert-images
```

### 5. Pemisahan Blok Download (Video MP4, Subtitle SRT & Torrent) dan Tombol Mandiri
Pada halaman detail film (`/movie/{slug}`), tautan unduhan ditata secara presisi dan dipisahkan menjadi 3 blok transparan agar tidak membingungkan pengguna:
1. **File Video Siap Nonton (Direct MP4 / Hardsub)**:
   - Tombol hijau terang: **"Download Video MP4"**.
   - Hanya menampilkan file video nyata (MP4/MKV) yang siap ditonton langsung di HP, PC, atau TV.
2. **File Subtitle Terpisah (SRT / VTT)**:
   - Tombol indigo: **"Download Subtitle"** lengkap dengan badge resmi SubDL / OpenSubtitles.
   - Mengatasi kerancuan sebelumnya di mana tautan subtitle sempat berlabel "Download MP4".
3. **File Torrent & Magnet Link (Video HD / 4K)**:
   - Tombol biru: **"Unduh Torrent"** yang terhubung langsung ke mesin pencari dan scraper torrent terverifikasi:
     - **Anime**: Nyaa.si (kategori anime HD batch) & AnimeTosho (Direct Torrents & DDL).
     - **Drama Asia / TV Series**: EZTV & 1337x.
     - **Hollywood / Movies**: 1337x & TorrentGalaxy (TGx).
4. **Tombol Mandiri "🔄 Cek / Segarkan Link" & "🔄 Cek / Segarkan Server"**:
   - Didesain presisi, menyatu rapi dengan header Pusat Unduhan dan bilah player streaming.
   - **Aturan Cooldown 24 Jam**: Jika URL dari sumber sama persis (sudah versi paling baru), tombol otomatis terkunci (*disabled*) selama **1 hari (24 jam)** secara persisten di browser (`localStorage`) untuk mencegah spam request ke sumber.

### 6. Scraping Film Barat & Domain Publik
```powershell
# Scrape film rilis tahun tertentu dari provider (tmdb / archive / yts / all)
.\cinembrot.exe -by-year 2024 -pages 1 -source yts

# Cari & ingest metadata film berkualitas tinggi langsung dari TMDb REST API
.\cinembrot.exe -tmdb "Inception" -year 2010

# Ingest film domain publik legal dari Internet Archive API
.\cinembrot.exe -archive -archive-limit 10

# Ingest film Creative Commons dari Blender Studio (Resolusi 4K)
.\cinembrot.exe -openmovies
```

> ⚠️ **PENTING — KEBIJAKAN KEAMANAN DATABASE:**
> Sesuai aturan server di `AGENTS.md` (Poin 4), seluruh operasi database yang memodifikasi skema atau data (`delete`, `restore`, `edit`) **WAJIB meminta izin (permission)** eksplisit dari pengguna terlebih dahulu sebelum dieksekusi.

---

## 📁 Struktur Direktori Proyek

```text
cinembrot/
├── auth/               # Modul autentikasi CMS Admin (HMAC-SHA256 & Session Cookie)
├── config/             # Konfigurasi aplikasi, env, database & API keys
├── database/           # Koneksi MariaDB, migration skema, & pengelolaan system_settings
├── i18n/               # Kamus translasi dwibahasa (Bahasa Indonesia & English)
├── imageprocessor/     # Konversi & kompresi WebP (Original & Thumbnail) hemat 90% storage
├── model/              # Definisi struct model GORM (Movie, DownloadLink, StreamLink, dll.)
├── pipeline/           # Pipeline agregasi, resolusi metadata, enricher, & perbaikan judul
├── provider/           # Adapter client penyedia data & streaming
│   ├── archive/        # Client API Internet Archive
│   ├── embed/          # Multi-server streaming embed generator (VidSrc, AutoEmbed, 2Embed, dll.)
│   ├── jikan/          # Client API MyAnimeList / Jikan v4 (Anime resmi)
│   ├── omdb/           # Client API OMDb (Rating IMDb)
│   ├── openmovies/     # Client film Creative Commons Blender Studio
│   ├── subtitles/      # Generator link pencarian subtitle dwibahasa (SubDL & OpenSubtitles)
│   ├── tmdb/           # Client REST API The Movie Database (TMDb Movie & TV)
│   └── yts/            # Client REST API YTS (YIFY Torrents)
├── public/             # Aset statis (WebP uploads, logo, bendera SVG, favicon)
├── scheduler/          # Background worker otomatis & cron scraping periodik
├── scraper/            # Engine scraping Colly, HTML cleaner, deteksi non-Latin, & repository query
├── server/             # HTTP Web Server, middleware admin, dan route handlers
│   └── views/          # Template HTML dinamis Tailwind dengan live auto-reload (F5)
├── torrentmgr/         # Download manager torrent internal & pipeline hardsub FFmpeg
├── validator/          # Engine validasi kesehatan tautan unduhan
├── schema.sql          # SQL dump database MariaDB lengkap
├── INSTRUKSI-SERVER.md # Sumber kebenaran spesifikasi & riwayat pembaruan sistem server
└── README.md           # Panduan lengkap fitur, instalasi, dan perintah CLI aplikasi
```

---

## 📄 Lisensi
Didistribusikan untuk tujuan edukasi dan pemutaran film/anime domain publik serta lisensi promosi resmi.


