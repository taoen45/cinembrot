# Instruksi Server Homelab taoen45

Dokumen ini adalah sumber kebenaran untuk spesifikasi, tujuan, dan perubahan server.
**Wajib dibaca dulu** sebelum agent mana pun mengubah, memperbaiki, atau menambah fitur.
Setelah perubahan, **wajib memperbarui** bagian "Riwayat perubahan" dan daftar aplikasi/fitur di bawah.

- File ini: `/home/taoen45/INSTRUKSI-SERVER.md`
- Aturan Cursor (selalu aktif): `/home/taoen45/.cursor/rules/server-instruksi.mdc`
- Ringkas untuk agent: `/home/taoen45/AGENTS.md`

Jangan menulis password, kunci WireGuard, atau isi `.env` ke dokumen ini.

---

## Tujuan server

Homelab di **Intel NUC5i3RYH** untuk:

1. **Website CINEMBROT (Golang)** di `~/docker/html` — Docker service `app` port `:8080`, Caddy `:80` reverse-proxy ke Go. Publik via Cloudflare Tunnel.
2. **MariaDB (latest)** + Redis 7. **Golang tidak butuh PHP** (PHP/Laravel sudah dihapus).
3. **Media pribadi** (belum aktif): storage `/data/media` + profile Jellyfin.
4. **Akses aman**: SSH, UFW + DOCKER-USER, fail2ban, WireGuard **klien**, Cloudflare Tunnel.

Stack: **Caddy → Go (CINEMBROT) + MariaDB + Redis**.

---

## Hardware dan OS

| Komponen | Nilai | Catatan |
|---|---|---|
| Model | Intel NUC5i3RYH (2015) | Firmware ~2018 |
| CPU | Intel Core i3-5005U 2c/4t @ 2.00 GHz | Lemah untuk transcode video |
| RAM | 6.7 GiB + swap 4 GiB | Batasi MariaDB/Redis/container |
| Disk | Colorful SL500 447 GB SSD | LVM |
| Root | `/` 98–100 GB (`ubuntu-vg/ubuntu-lv`) | Jangan isi media di sini |
| Data | `/data` ~338–344 GB (`ubuntu-vg/data`) | Docker, media, unduhan, backup |
| GPU | Intel HD 5500 | Direct play saja |
| Hostname | `taoen45` | Ubuntu 24.04.4 LTS (noble), kernel 6.8 |
| User | `taoen45` (sudo, docker) | Sudo butuh password |

RAM ketat. Jangan menambah layanan berat tanpa menyesuaikan `mem_limit`. Upgrade RAM ke 16 GB (maks NUC ini) sangat membantu.

---

## Zona waktu dan pemeliharaan malam

- **Timezone wajib: `Asia/Jakarta` (WIB, UTC+7).** Sistem sempat `Etc/UTC`; jadwal subuh harus WIB.
- **03:40 WIB** — `apt update --fix-missing && apt full-upgrade -y` (noninteractive).
- **04:00 WIB** — reboot host. Timer menunggu jika apt masih jalan (maks ~4 menit).
- Unit systemd: `homelab-nightly-upgrade.timer`, `homelab-nightly-reboot.timer`.
- Skrip sumber: `~/docker/host/homelab-nightly-*.sh` + `*.service` + `*.timer`.
- Installer: `sudo bash /home/taoen45/docker/host/install-nightly-maintenance.sh`
- Log: `/var/log/homelab/nightly-upgrade.log`, `/var/log/homelab/nightly-reboot.log`
- `APT::Periodic::Unattended-Upgrade` dimatikan supaya tidak bentrok dengan jadwal 03:40.
- Docker `restart: unless-stopped` — setelah reboot, stack naik sendiri.

Cek jadwal: `systemctl list-timers 'homelab-nightly-*'`
Jangan reboot manual di jam sibuk kecuali diminta.

---

## Jaringan

| Interface | Peran | Status / IP |
|---|---|---|
| `wlp2s0` | WiFi utama saat ini | `192.168.5.115/24` (metric 600) |
| `enp0s25` | LAN Intel I218-V, DHCP otomatis | Prioritas metric 100; sering down jika kabel belum colok |
| `wg0` | WireGuard **klien** | `55.0.0.157/24`, endpoint `103.127.133.219:55555` |
| `lo` | MariaDB, Redis, health Go | `127.0.0.1` |

- Netplan LAN: `~/docker/host/60-ethernet.yaml` → `/etc/netplan/60-ethernet.yaml`
- LAN: `192.168.5.0/24`. Boot tidak boleh macet jika kabel dicabut (`optional: true`).
- WireGuard di **host**, bukan Docker. Mode **klien** (tidak ada `ListenPort`). Config: `~/wireguard/client.conf`, apply: `sudo bash ~/wireguard/apply-as-client.sh`
- Port server WG `51820/udp` tidak lagi dibuka.
- Firewall DOCKER-USER masih mengizinkan `10.8.0.0/24` (sisa mode server lama) + LAN + docker bridges.

---

## Keamanan

- UFW: default deny incoming, allow outgoing, allow SSH.
- HTTP/HTTPS dimaksudkan hanya LAN (+ VPN). Jangan buka 80/443 ke internet tanpa Cloudflare/Zero Trust.
- `DOCKER-USER`: drop selain localhost, `172.16.0.0/12`, `192.168.5.0/24`, `55.0.0.0/24`, `10.8.0.0/24`, `wg0`, `br+`, `docker0`.
- fail2ban SSH aggressive; ignore LAN + `55.0.0.0/24` (+ kompatibel `10.8.0.0/24`).
- MariaDB bind `0.0.0.0` untuk remote via WireGuard; wajib batasi akses di firewall hanya subnet WG. Redis tetap bind `127.0.0.1` + password.
- `cloudflared` 2026.8.3: layanan systemd `cloudflared.service` (token di `/etc/cloudflared/token`, jangan dibuka/disalin ke chat).
- Tunnel terhubung ke Cloudflare. Public hostname dikelola di dashboard Zero Trust, bukan file lokal.

---

## Docker dan aplikasi

Direktori: `/home/taoen45/docker`  
Compose: `~/docker/compose.yml`  
Env (rahasia): `~/docker/.env` — `TZ=Asia/Jakarta`  
Docker data-root: `/data/docker`  
DNS daemon: `192.168.5.1`, `1.1.1.1`  
Build image: pakai `network: host` (DNS/IPv6 di bridge sering gagal).

### Aktif (default)

| Service | Image / build | Bind | RAM / CPU |
|---|---|---|---|
| `app` | build `~/docker/html` (CINEMBROT Go) | host `:8080` | 768m / 1.5 |
| `mysql` | `mariadb:latest` | host `:3306` (akses WG via UFW) | 768m / 1.0, InnoDB buffer 256M |
| `redis` | `redis:7-alpine` | host `127.0.0.1:6379` | 320m / 0.5, maxmemory 256mb |
| `caddy` | `caddy:2-alpine` | host `:80` → `127.0.0.1:8080` | 128m / 0.25 |
| `cloudflared` | paket host | outbound tunnel saja | ~16m |

Kode website: `~/docker/html` (module `cinembrot`). Env app: `~/docker/html/.env`.  
**DB_HOST wajib `127.0.0.1`** (MariaDB `network_mode: host` — jangan pakai hostname service `mysql`).  
DB name/user/pass mengikuti `html/.env` (bukan hanya `docker/.env`). App auto-migrate + buat DB jika perlu.  
Cloudflare Public hostname: Path **kosong**, URL `http://localhost:80` (Caddy meneruskan ke Go).

**Golang tidak membutuhkan PHP.** Folder `docker/php` dan profile Laravel sudah dihapus.

Tool Go di host (opsional, development): `sudo bash ~/docker/setup-golang.sh` lalu `cd ~/docker/html && go run . -serve`.

### Tidak aktif (profile)

| Service | Profile | Kapan dinyalakan |
|---|---|---|
| `jellyfin` | `media` | Koleksi pribadi; media read-only `/data/media` |

### Storage `/data`

- `/data/docker` — data Docker
- `/data/media/anime`, `/data/media/movies`
- `/data/downloads`
- `/data/backups`

---

## Perintah operasional

```bash
cd /home/taoen45/docker
docker compose ps
docker compose up -d --build
docker compose logs -f --tail=100
./status.sh
```

Laravel (sudah dihapus — jangan aktifkan PHP):

```bash
# PHP/Laravel tidak dipakai. Website = Go CINEMBROT.
```

Cloudflare Tunnel:

```bash
sudo systemctl status cloudflared
sudo journalctl -u cloudflared -n 50 --no-pager
```

Caddy setelah ubah Caddyfile:

```bash
cd /home/taoen45/docker
docker compose up -d caddy --force-recreate
curl -sS -o /dev/null -w "%{http_code}\n" http://127.0.0.1/
curl -sS -o /dev/null -w "%{http_code}\n" http://127.0.0.1:8080/
```

Build ulang website Go:

```bash
cd /home/taoen45/docker
docker compose up -d --build app caddy
```

Pasang Go di host (dev):

```bash
sudo bash /home/taoen45/docker/setup-golang.sh
```

Perintah CLI / Terminal (Setara Artisan di Laravel):

```bash
cd /home/taoen45/docker/html

# Scrape anime resmi dari MyAnimeList (Jikan API)
go run . -scrape-anime -anime-cat top -anime-limit 15
go run . -scrape-anime -anime-cat seasonal -anime-limit 20

# Scrape drama Asia dari TMDb TV API (K-Drama, C-Drama, J-Drama, Thai)
go run . -scrape-drama -drama-lang ko -drama-pages 1
go run . -scrape-drama -drama-lang zh -drama-pages 1

# Pengecekan & validasi link download di database
go run . -check-links

# Jalankan 1 siklus auto-scraper semua tahun
go run . -auto-scrape
```

Jellyfin (hanya jika diminta):

```bash
cd /home/taoen45/docker
docker compose --profile media up -d
```

Setup host awal (sudah pernah dijalankan): `sudo bash ~/docker/setup-server.sh`  
Pasang Go toolchain di host (dev): `sudo bash ~/docker/setup-golang.sh`

---

## Domain dan Cloudflare Zero Trust

Tunnel sudah jalan di host. Routing hostname diubah di **dashboard**, bukan di NUC.

Dashboard: Zero Trust → **Networks** → **Tunnels** → tunnel NUC → **Public hostname**.

Aturan: kolom **Path** adalah path URL (contoh `/api`), **bukan** folder disk. Folder situs = `~/docker/html`, disajikan Caddy di `http://localhost:80`.

### Ganti ke domain utama `cinembrot.my.id` (bisa)

1. Tambah Public hostname:
   - Subdomain: `@` atau kosong (apex) + Domain: `cinembrot.my.id`
   - Type: HTTP
   - URL: `http://localhost:80`
   - Path: **kosong**
2. Simpan. Cloudflare biasanya membuat DNS proxied untuk apex.
3. Opsional: hostname `www.cinembrot.my.id` dengan URL yang sama.
4. `test.cinembrot.my.id` boleh **tetap** (subdomain uji) atau dihapus jika tidak dipakai.

Tidak perlu ganti folder di server. Domain utama dan `test` bisa merujuk ke origin yang sama (`localhost:80`); Caddy yang membedakan isi.

### Tambah folder `test` (sudah disiapkan)

Folder: `~/docker/html/test/` (file `index.html`).

Dua cara akses (boleh keduanya):

**A. Path** — `https://test.cinembrot.my.id/test/` atau nanti `https://cinembrot.my.id/test/`  
Tidak perlu hostname baru. Caddy `handle_path /test*` mengarah ke `html/test`.

**B. Subdomain terpisah** (opsional, nanti) — misalnya `staging.cinembrot.my.id`  
Tambah Public hostname di dashboard (Path kosong, URL `http://localhost:80`), lalu tambah blok `http://staging.cinembrot.my.id` di `Caddyfile` yang `root` ke folder khusus. Jangan arahkan `test.cinembrot.my.id` ke folder `test` selama subdomain itu masih dipakai sebagai situs publik.

Tambah folder lain (contoh `blog`): buat `~/docker/html/blog/`, copy pola `/test` di `~/docker/caddy/Caddyfile`, recreate Caddy. Untuk subdomain `blog.cinembrot.my.id`, tambah Public hostname di dashboard (Path kosong, URL `http://localhost:80`).

---

## Batasan dan larangan

- **Urusan Database**: Ekstra hati-hati. Operasi `delete`, `restore`, `edit`, migrasi struktur, atau modifikasi data **WAJIB meminta izin (permission)** dari user terlebih dahulu. Dilarang melakukan DROP tabel atau truncate data sembarangan.
- **Bahasa**: Selalu menggunakan Bahasa Indonesia dalam percakapan, instruksi, dan dokumentasi.
- Jangan membuka port database ke internet/LAN; izinkan hanya dari subnet WireGuard tepercaya.
- Jangan mengaktifkan Jellyfin tanpa permintaan user.
- Jangan menyiapkan situs unduh film berhak cipta untuk publik.
- Jangan commit `.env`, kunci WireGuard, atau password.
- Hormati `mem_limit`; NUC ini mudah kehabisan RAM.
- Transcode 1080p di i3-5005U hampir pasti tersendat.
- Perubahan host (timezone, UFW, systemd, apt, WireGuard) butuh sudo + update dokumen ini.

---

## Cara agent wajib bekerja

1. Baca file ini dan `AGENTS.md` sebelum merencanakan perubahan.
2. Selalu gunakan Bahasa Indonesia dalam setiap interaksi dan dokumentasi.
3. Untuk urusan database (terutama `delete`, `restore`, `edit` data atau skema): **Wajib meminta izin eksplisit dari user** sebelum eksekusi.
4. Jika tidak paham, tanya user — jangan mengarang tujuan.
5. Ubah sesedikit mungkin; jangan merombak stack yang sudah jalan.
6. Setelah menambah aplikasi, fitur, cron/timer, port, atau profil Docker:
   - tambahkan ke daftar di atas
   - isi **Riwayat perubahan**
7. Jangan hapus riwayat lama; tambah baris baru.
8. Jika mengubah file instruksi/agent (`INSTRUKSI-SERVER.md`, `AGENTS.md`, aturan Cursor), wajib catat juga di **Riwayat perubahan**.

---

## Riwayat perubahan

| Tanggal | Perubahan |
|---|---|
| 2026-09-02 | Setup awal: Ubuntu 24.04, LAN DHCP, Docker, Go+MySQL+Redis+Caddy (sebelum migrasi MariaDB), UFW, fail2ban, cloudflared, `/data` LVM. |
| 2026-09-02 | Percobaan Laravel/PHP; dikembalikan ke Go. `html/` dikosongkan sebagai cadangan. PHP profile `laravel`. |
| 2026-09-02 | WireGuard diubah dari server (`10.8.0.1`) menjadi **klien** `55.0.0.157`. |
| 2026-09-03 | Timezone `Asia/Jakarta`. Update apt `03:40` + reboot `04:00` via systemd timer. Unattended-upgrade otomatis dimatikan. Dokumen + aturan Cursor wajib-baca dibuat. |
| 2026-09-03 | Cloudflare Tunnel aktif. `test.cinembrot.my.id` → `http://localhost:80`. Caddy sajikan `~/docker/html` (bukan Go). Path CF harus kosong (bukan path folder). |
| 2026-09-03 | Folder `~/docker/html/test` via path `/test`. Domain utama `cinembrot.my.id` ditambah di dashboard (Path kosong, URL `http://localhost:80`). **Go tidak butuh PHP.** |
| 2026-09-03 | Database (saat itu masih MySQL) direvisi agar bisa remote dari WireGuard: bind ke `0.0.0.0`, firewall diarahkan hanya izinkan subnet WG (`55.0.0.0/24`) ke port 3306. |
| 2026-09-03 | Kredensial root database dirotasi sesuai permintaan user, file `.env` disinkronkan, dan verifikasi login root berhasil (nilai password tidak dicatat di dokumen). |
| 2026-09-03 | Aturan agent/instruksi diperketat: setiap perubahan pada file instruksi/agent wajib dicatat ke **Riwayat perubahan**. |
| 2026-09-03 | Database service dimigrasikan dari `mysql:8.4` ke `mariadb:latest` dengan volume baru `mariadb_data` agar data MySQL lama tetap tersimpan untuk rollback. |
| 2026-09-03 | Pembersihan istilah dokumentasi: referensi MySQL lama diberi konteks historis, state aktif ditegaskan sebagai MariaDB. |
| 2026-09-03 | Rule UFW aktif diterapkan: allow `3306/tcp` dari subnet WireGuard `55.0.0.0/24` (MariaDB WireGuard only). |
| 2026-09-04 | Website CINEMBROT Go di `~/docker/html` jadi service `app` (Docker). Caddy proxy ke `:8080`. `DB_HOST=127.0.0.1`. PHP/Laravel dihapus. Skrip `setup-golang.sh` untuk toolchain host. |
| 2026-09-12 | Aturan diperketat: proteksi database wajib izin user (delete, restore, edit), kewajiban komunikasi Bahasa Indonesia, penegasan pencatatan audit perubahan, dan sinkronisasi dokumentasi proyek CINEMBROT (arsitektur paket & opsi CLI). |
| 2026-09-12 | Verifikasi koneksi database ke MariaDB 12.3.3 di `55.0.0.157:3306` via WireGuard berhasil. `.env` dikonfirmasi: `DB_HOST=55.0.0.157`, `DB_PORT=3306`, `DB_NAME=cinembrot`. Kolom `movies.type` (`varchar(50) DEFAULT 'movie'`) terkonfirmasi sudah ada di database live — tidak ada ALTER TABLE. |
| 2026-09-12 | **Fitur Baru: Kategori Anime & Drama Pendek.** Perubahan mencakup: (1) Template publik baru `server/views/anime.html` dan `server/views/drama_pendek.html` dengan desain Hero banner dan filter bar. (2) Navigasi publik di `server/views/layout.html` — menu Anime/Drama Pendek di navbar desktop dan mobile pill navbar. (3) Sidebar CMS Admin di `server/views/admin_layout.html` — menu "Kelola Anime" dan "Drama Pendek". (4) Dropdown tipe di `server/views/admin_movie_form.html` — pilihan `movie`, `anime`, `drama_pendek`, `series`. (5) Tabel film CMS (`server/views/admin_movies.html`) — mendukung badge tipe dan tombol tambah sesuai konteks. (6) Dashboard (`server/views/admin_dashboard.html`) — kartu statistik Anime & Drama Pendek. (7) Backend Go: handler publik `HandleAnime`/`HandleDramaPendek` di `server/handlers.go`. (8) Handler admin `HandleAdminAnime`/`HandleAdminDramaPendek` + field `TypeContext` di `AdminPageData` + statistik `total_anime`/`total_drama_pendek` di dashboard + support type pada form create/edit di `server/admin_handlers.go`. (9) Rute HTTP baru di `server/server.go`: `GET /anime`, `GET /drama-pendek`, `GET /admin/anime`, `GET /admin/drama-pendek`. (10) Template anime.html dan drama_pendek.html didaftarkan di `loadTemplates()`. |
| 2026-09-14 | Persiapan dan kompilasi lokal Windows: instalasi Go toolchain di host Windows, build `cinembrot.exe`, verifikasi konektivitas database remote `55.0.0.157:3306` via WireGuard, dan verifikasi web server berjalan di port 8080. Commit dan push seluruh pembaruan fitur Anime & Drama Pendek beserta dokumentasi ke repositori. |
| 2026-09-15 | **Fasilitas Scraper Resmi Anime & Drama Asia + Integrasi Subtitle:** (1) Modul baru `provider/jikan` untuk scraping Anime MyAnimeList resmi (Top Anime & Seasonal). (2) Modul `provider/subtitles` untuk generasi otomatis tautan unduh dan kandidat subtitle dwibahasa (Bahasa Indonesia & English). (3) Perluasan `provider/tmdb` dengan endpoint Discover TV & detail serial drama Asia (K-Drama, C-Drama, J-Drama, Thai Drama). (4) Penambahan pipeline `IngestAnime()` dan `IngestAsianDramas()` dengan pemrosesan poster WebP. (5) Penambahan rute/handler CMS Admin `POST /admin/tools/scrape-anime` dan `POST /admin/tools/scrape-drama` serta kartu eksekusi di `admin_tools.html`. |
| 2026-09-15 | **Penambahan Perintah CLI Terminal (Setara Artisan Laravel) & Optimasi WebP:** (1) Menambahkan flag CLI `-scrape-anime` (argumen `-anime-cat` dan `-anime-limit`) serta `-scrape-drama` (argumen `-drama-lang` dan `-drama-pages`) di `main.go` sehingga scraping dapat dijalankan langsung via terminal. (2) Optimasi kompresi WebP (80%/75%) dan pembatasan lebar resolusi gambar di `imageprocessor/processor.go` agar hemat storage s/d 90%. (3) Dokumentasi perintah operasional terminal disinkronkan di `README.md` dan `INSTRUKSI-SERVER.md`. |
| 2026-09-15 | **Perbaikan Gambar Broken & Fallback Scraper:** (1) Mengatasi gambar broken/retak akibat file `.webp` belum ada di lokal dengan menambahkan Smart Fallback di handler `GET /uploads/` (`server/server.go`) yang menyajikan `poster-placeholder.svg` dan atribut `onerror` fallback di template web (`home.html`, `anime.html`, `drama_pendek.html`, `list.html`, `detail.html`). (2) Menambahkan fallback scraper Anime resmi TMDb (`with_genres=16&with_original_language=ja`) di `provider/tmdb/tmdb.go` & `pipeline/pipeline.go` jika API Jikan/MyAnimeList mengalami rate limit/504. (3) Pengujian scraper anime (`-scrape-anime`) dan drama asia (`-scrape-drama`) diverifikasi 100% sukses menyimpan data, poster WebP, dan link subtitle dwibahasa ke MariaDB. |
| 2026-09-15 | **Fitur Terjemahan Multi-Bahasa (Indonesia & English):** (1) Modul baru `i18n/i18n.go` berisi kamus kamus lengkap dwibahasa (Bahasa Indonesia sebagai default/utama dan English sebagai bahasa kedua) untuk seluruh elemen antarmuka, navigasi, filter, katalog, detail film, dan footer. (2) Language Switcher dropdown elegan di navbar desktop (`🇮🇩 ID` / `🇬🇧 EN`) dan quick switcher di mobile sub-navbar. (3) Rute baru `GET /set-lang?lang=en` dengan Set-Cookie persisten (1 tahun) dan auto-redirect. (4) Template engine helper `{{t $.Lang key}}` terdaftar di `server/server.go` dan field `Lang` terpasang di seluruh handler publik `server/handlers.go`. |
| 2026-09-15 | **Penyempurnaan Translasi Total & Bendera SVG:** (1) Mengganti teks emoji bendera dengan file grafis SVG asli berwarna (`/img/flags/id.svg` dan `/img/flags/en.svg`) agar bendera Merah Putih dan Union Jack tampil tajam dan berwarna di semua sistem operasi dan browser Windows. (2) Menambahkan method `GetEnglishSynopsis` di `provider/tmdb/tmdb.go` dengan cache memori dan integrasi ke `HandleMovieDetail` & `HandleHome` di `server/handlers.go` sehingga sinopsis film langsung berubah total ke Bahasa Inggris resmi saat memilih English. (3) Mengintegrasikan engine Google Website Translator di client-side (`layout.html`) untuk translasi otomatis penuh seluruh elemen halaman. |
| 2026-09-15 | **Pemasangan 5 Unit Iklan Adsterra & Halaman Pengaturan CMS Dinamis (/admin/settings):** (1) Integrasi penuh 5 unit iklan Adsterra: Popunder, Social Bar, Banner 728x90 Header, Native Banner Rekomendasi, dan Direct Smartlink tombol fast streaming/download di `layout.html` dan `detail.html`. (2) Pembuatan halaman backend baru `/admin/settings` (`server/views/admin_settings.html` & `server/admin_settings_handlers.go`) untuk kontrol dinamis seluruh fitur: master switch iklan ON/OFF, input script masing-masing unit iklan, saklar pengalih bahasa ID/EN, saklar Google Translate engine, saklar kolom komentar penonton, dan identitas/branding situs (nama brand & tagline). (3) Penambahan helper pengaturan dan seeding default di `database/mariadb.go` (`GetAllSettings`, `GetSetting`, `SaveSetting`, `seedDefaultSettings`) yang menyimpan seluruh konfigurasi ke tabel `system_settings` di MariaDB secara aman dan persisten tanpa migrasi struktur tabel. (4) Otomatisasi injeksi pengaturan ke `PageData` via helper `PopulatePageData` di `server/handlers.go` dan `server/server.go`. |
| 2026-09-15 | **Penambahan Masif ~100 Anime Populer & Terbaru + Tracking Poster WebP ke Git:** (1) Pembaruan pipeline `IngestAnime()` (`pipeline/pipeline.go`) dengan mekanisme multi-page pagination dan kombinasi anime terpopuler sepanjang masa (*Attack on Titan*, *Death Note*, *Fullmetal Alchemist*, *One-Punch Man*, *Demon Slayer*, *Frieren*, dll.) serta anime rilis terbaru/musim 2025-2026 (*Solo Leveling*, *Mushoku Tensei S3*, *Tanya the Evil II*, dll.). (2) Sebanyak 91 judul anime unik baru berhasil di-scrape, diperkaya metadatanya, dibuatkan link subtitle dwibahasa (ID/EN), dan disimpan ke database MariaDB. (3) Seluruh poster dan thumbnail terkompresi efisien dalam format WebP lokal di `public/uploads/` dan dimasukkan ke pelacakan Git (dihapus dari `.gitignore`) serta di-push ke repositori GitHub fork (`fork/main`). |
| 2026-09-15 | **Maksimalisasi Iklan Adsterra (5 Banner Baru: Sidebars, Mobile Sticky & Rectangle):** (1) Penambahan unit banner Adsterra baru: Floating Sticky Skyscraper Kiri 160x600 (`fixed left-2 top-24`) & Kanan 160x300 (`fixed right-2 top-24`) khusus layar 2XL desktop dengan tombol tutup (dismiss), Mobile Sticky Bottom 320x50 khusus smartphone dengan tombol tutup, Pre-footer Banner 468x60, dan Medium Rectangle 300x250 di halaman detail film sebelum komentar penonton. (2) Penambahan seeding default dan pembantu konfigurasi di `database/mariadb.go` (`system_settings`) serta injeksi otomatis ke `PageData` (`server/handlers.go`). (3) Penyediaan 5 slot input baru di CMS Admin `/admin/settings` (`server/views/admin_settings.html` & `server/admin_settings_handlers.go`) agar pemilik situs leluasa mengganti atau menonaktifkan kode banner kapan saja. |
| 2026-09-15 | **Optimasi Desain Versi Mobile & Perbaikan UX Smartphone:** (1) Memperbaiki bug horizontal scroll overflow di HP dengan menyembunyikan banner 728x90 (`hidden md:block`) dan banner 468x60 (`hidden sm:block`) agar halaman terkunci pas di layar HP tanpa goyang ke samping. (2) Menambahkan bantalan safe area `pb-24 md:pb-6` pada kontainer utama (`layout.html`) agar floating banner 320x50 tidak menutupi tombol dan konten bawah. (3) Menambahkan gesture Touch Swipe (`touchstart`, `touchend`) pada slider hero beranda (`home.html`) untuk perpindahan slide alami dengan usapan jempol, serta menyembunyikan tombol panah di layar HP. (4) Merampingkan grid katalog (`gap-3 sm:gap-4`) dan wrapping badge metadata (`flex-wrap`) di halaman `anime.html` dan `drama_pendek.html`. (5) Membuat tombol aksi download MP4 matang dan tombol kirim komentar di `detail.html` responsif `w-full sm:w-auto` agar nyaman diakses oleh pengguna smartphone. |

