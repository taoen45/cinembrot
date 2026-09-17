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
- **05:00 WIB & 17:00 WIB** — Auto-Scraper Harian CINEMBROT: memindai seluruh rilis film baru, anime on-going/seasonal, drama Asia (Korea, China, Jepang, Thailand), dan rilis episode baru tahun berjalan (2026). Dijalankan otomatis oleh internal clock watcher container Go `app` dan dijadwalkan via crontab user `taoen45` (`~/docker/host/cinembrot-daily-scrape.sh`).
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

Bisa dijalankan via Docker container (tanpa perlu Go di host) atau langsung via host jika Go terpasang:

```bash
# --- METODE A: VIA DOCKER COMPOSE (DIREKOMENDASIKAN) ---
cd /home/taoen45/docker

# Scrape Anime / Drama Asia / Hollywood rilis 2026
docker compose exec app ./cinembrot -scrape-anime -year 2026
docker compose exec app ./cinembrot -scrape-drama -year 2026
docker compose exec app ./cinembrot -scrape-hollywood -year 2026

# Isi link download massal (Torrent & Subtitle) & server streaming embed
docker compose exec app ./cinembrot -populate-downloads
docker compose exec app ./cinembrot -populate-streams

# Perbaikan judul non-Latin (Kanji/CJK) & auto-translate sinopsis dwibahasa
docker compose exec app ./cinembrot -fix-titles
docker compose exec app ./cinembrot -translate-synopsis

# Validasi kesehatan tautan & konversi poster ke WebP lokal
docker compose exec app ./cinembrot -check-links
docker compose exec app ./cinembrot -convert-images

# --- METODE B: LANGSUNG DI HOST DENGAN GO RUN / BINARY ---
cd /home/taoen45/docker/html

go run . -scrape-anime -year 2026
go run . -scrape-drama -year 2026
go run . -scrape-hollywood -year 2026
go run . -populate-downloads
go run . -populate-streams
go run . -fix-titles
go run . -translate-synopsis
go run . -check-links
go run . -convert-images
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
### SOP 3 Menit: Cara Cepat Mengganti Domain Jika Terkena Blokir (Anti-Banned)

Jika domain yang sedang dipakai terkena blokir Trust Positif / Nawala / Kominfo:

1. **Beli Domain Baru** di registrar mana saja (misal: `cinembrot-baru.com` atau `cinembrot.vip`).
2. **Arahkan ke Cloudflare**:
   - Tambahkan domain ke akun Cloudflare.
   - Di menu **Zero Trust** → **Networks** → **Tunnels** → pilih tunnel server ini → **Public hostname**:
     - Klik **Add a public hostname**.
     - Domain: `cinembrot-baru.com` (Path: **kosong**).
     - Type: `HTTP`, URL: `http://localhost:80`.
     - Klik **Save hostname**.
3. **Selesai Seketika (Zero Downtime)**:
   - Karena Go CINEMBROT telah dilengkapi **Auto-Detect Domain Mode** di `GetSiteURL()`, website akan **LANGSUNG berjalan normal dan mengenali domain baru tersebut secara otomatis**, termasuk seluruh canonical link, sitemap, OpenGraph, dan streaming player tanpa perlu restart server atau utak-atik tabel database!
   - (Opsional) Buka CMS `/admin/settings` untuk mengaktifkan **Banner Notifikasi Pindah Domain** agar penonton di domain lama mengetahui alamat domain baru.

---

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
| 2026-09-15 | **Optimasi Gitignore & Pembersihan 4.000+ Untracked Files:** Memperbarui `.gitignore` dengan menambahkan `public/uploads/` serta file deployment Docker lokal (`Dockerfile`, `.dockerignore`, `.gitkeep`). Hal ini mencegah ribuan foto aktor, backdrop hasil unduhan dinamis scraper, dan file konfigurasi lokal membanjiri status Git/VS Code, sementara 1.030 file poster/thumbnail yang telah ter-commit sebelumnya tetap aman terlacak di Git. |
| 2026-09-15 | **Fitur Streaming Video Multi-Server & Episode Switcher (Anime, Drama Pendek & Movies):** (1) Pembuatan modul baru `provider/embed/embed.go` untuk generasi otomatis 5 embed server video streaming (Server 1: VidSrc HD, Server 2: AutoEmbed Fast, Server 3: 2Embed VIP, Server 4: VidLink Pro, Server 5: SuperEmbed Multi, plus Trailer Resmi YouTube). (2) Tampilan player video interaktif di `server/views/detail.html`: dilengkapi tombol pemilih server streaming (Server Switcher Bar) dan navigasi episode (Episode Selector Bar) yang responsif dan berganti instan via JavaScript tanpa reload halaman. (3) Resolusi otomatis TMDb ID berbasis pencarian TV/Movie di `server/handlers.go`. (4) Integrasi auto-embed pada scraper `IngestAnime()` dan `IngestAsianDramas()` di `pipeline/pipeline.go`. (5) Perintah CLI baru `go run . -populate-streams` di `main.go` yang berhasil mengisi server streaming untuk 274 judul film/anime/drama ke tabel `stream_links` MariaDB. |
| 2026-09-15 | **Judul Versi English QWERTY, Auto-Fill Sinopsis & Scraper Berdasarkan Tahun Rilis:** (1) Resolusi otomatis judul non-Latin (Kanji, Kana, Hanzi, Hangul) ke judul resmi English standar keyboard QWERTY via TMDb Translations API (`iso_639_1 == "en"`) dan Jikan `TitleEnglish`/Romaji di `provider/tmdb/tmdb.go` dan `provider/jikan/jikan.go`, nama asli disimpan di `original_title` dan `alternative_titles` (alias name). Contoh: `異世界かるてっと` berubah menjadi `Isekai Quartet` dan URL slug bersih `isekai-quartet-2019`. (2) Pengisian otomatis sinopsis kosong saat scraping dan penanganan di `scraper/repository.go` agar tidak menimpa sinopsis yang sudah ada dengan string kosong. (3) Penambahan opsi filter tahun rilis pada scraper: `-scrape-anime -year 2026` dan `-scrape-drama -year 2026` via endpoint TMDb Discover TV `first_air_date_year` dan Jikan `start_date`/`end_date`. (4) Perintah utilitas CLI baru `go run . -fix-titles` di `main.go` dan `pipeline/pipeline.go` yang berhasil menyisir dan memperbarui 25 judul kanji lama dan sinopsis kosong di database MariaDB dengan izin user. |
| 2026-09-16 | **Fitur Baru: Film Hollywood / Box Office & Scraper Resmi TMDb Movie API:** (1) Halaman katalog publik baru `/hollywood` (dan alias `/box-office`) dengan template modern `server/views/hollywood.html` (Hero banner, filter genre, tahun rilis, rating, dan pengurutan). (2) Navigasi menu Hollywood terpasang di navbar desktop dan sub-navbar pill mobile (`server/views/layout.html`). (3) Integrasi scraper Film Hollywood resmi via TMDb Movie API (`DiscoverHollywoodMovies` di `provider/tmdb/tmdb.go` dan `IngestHollywoodMovies` di `pipeline/pipeline.go`) lengkap dengan pembuatan poster WebP lokal, link subtitle dwibahasa (ID/EN), dan 5 server streaming video embed (VidSrc, AutoEmbed, 2Embed, VidLink, SuperEmbed). (4) Perintah CLI terminal baru `-scrape-hollywood` dengan opsi `-hollywood-cat`, `-hollywood-pages`, dan `-year` di `main.go`. (5) Integrasi CMS Admin: menu sidebar "Kelola Hollywood" (`/admin/hollywood`) dan kartu eksekusi scraper Hollywood di `/admin/tools` (`POST /admin/tools/scrape-hollywood`). (6) Terdaftar resmi di Dynamic XML Sitemap (`/sitemap.xml`) dengan prioritas 0.9. |
| 2026-09-16 | **Peningkatan Halaman Katalog & Filter Lengkap (/filter) & Analisis Command Operasional:** (1) Penambahan input pencarian multi-kolom kata kunci (`q`/`keyword`) yang memeriksa judul film (`title`), judul asli (`original_title`), nama alias/alternatif (`alternative_titles`), dan sinopsis. (2) Penambahan filter Jenis Film/Tipe Konten (`type`: Hollywood/Box Office, Anime, Drama Pendek, Bioskop/Movie, Series) dan filter Rating Minimal (`rating`: 8.0+, 7.0+, 6.0+, 5.0+). (3) Perbaruan antarmuka `server/views/list.html` dan `server/views/home.html` dengan tata letak grid modern yang responsif serta perbaikan pagination link yang mempertahankan seluruh query filter (`/filter?p=...&q=...&type=...&rating=...`). (4) Klarifikasi dokumentasi CLI terminal di `README.md` mengenai status perintah operasional web server (`-serve`), scheduler worker (`-daemon`), dan legacy cycle (`-auto-scrape`). |
| 2026-09-16 | **Auto-Translate Sinopsis Dwibahasa (Opsi A: synopsis_en) & Scraper Refresh Broken Download Link:** (1) Penambahan field & kolom baru `synopsis_en` (`LONGTEXT`) di tabel `movies` MariaDB dengan persetujuan user (Opsi A). (2) Modul baru `translator/translator.go` yang otomatis mendeteksi bahasa sumber dan menerjemahkan sinopsis ke Bahasa Indonesia (`synopsis`) dan Bahasa Inggris (`synopsis_en`) menggunakan Chrome Client translator. (3) Integrasi auto-translate pada seluruh pipeline scraping (Anime, Drama Asia, Hollywood) dan perintah CLI baru `.\cinembrot.exe -translate-synopsis` untuk menyisir film yang sudah ada di database. (4) Fitur scraper link download mandiri & endpoint `POST /api/movie/{id}/refresh-download`: tombol interaktif di `detail.html` untuk memeriksa kesehatan link download dan melakukan rescrape otomatis ke website sumber jika link mati, serta auto-disable jika URL dari sumber sama persis. |
| 2026-09-16 | **Integrasi URL Link Download Otomatis pada Seluruh Scraper & Perintah Massal (-populate-downloads):** (1) Menjamin bahwa seluruh command scraping (`-scrape-anime`, `-scrape-drama`, `-scrape-hollywood`) 100% meng-include URL link download dan subtitle dwibahasa secara otomatis saat scraping film baru. (2) Penambahan fitur CLI terminal `.\cinembrot.exe -populate-downloads` (`pipeline.PopulateAllExistingDownloadLinks`) untuk menyisir seluruh film di MariaDB yang belum memiliki link download dan otomatis melengkapinya dari sumber torrent resmi (YTS 720p/1080p/4K) dan subtitle (SubDL/Subsource). |
| 2026-09-16 | **Pembaruan Dokumentasi Operasional Server & Verifikasi Runtime:** (1) Penambahan panduan lengkap cara menyalakan (Foreground & Background), mematikan (`Ctrl+C` & `Stop-Process`), serta memeriksa status server di `README.md`. (2) Verifikasi runtime: web server terkonfirmasi berjalan 100% normal dan menyajikan HTML di `http://localhost:8080`. |
| 2026-09-16 | **Pemisahan Blok Download (Video MP4 vs Subtitle vs Torrent), Desain Tombol Presisi & Generator Torrent:** (1) Perbaikan tombol "Cek / Segarkan Link" di header Pusat Unduhan `detail.html` agar presisi, simetris, dan menyatu rapi dengan border box. (2) Pemisahan modul dan tampilan unduhan menjadi 3 bagian terpisah: File Video Siap Nonton (Direct MP4 / Hardsub) dengan tombol hijau "Download Video MP4", File Subtitle Terpisah (SRT / VTT) dengan tombol indigo "Download Subtitle", dan File Torrent & Magnet Link dengan tombol biru "Unduh Torrent". Mengatasi bug di mana tautan subtitle sebelumnya bertuliskan "Download MP4". (3) Pembuatan modul baru `provider/torrents/torrents.go` yang menghasilkan link torrent terverifikasi: Nyaa.si & AnimeTosho untuk Anime, EZTV & 1337x untuk Drama Asia/Series, serta 1337x & TorrentGalaxy untuk film Hollywood. (4) Integrasi auto-generate torrent candidates di `HandleMovieDetail` dan pipeline scraping. |
| 2026-09-16 | **Pembersihan Syntax Error & Linter Editor pada Template detail.html:** (1) Menghapus blok komentar HTML lama `<!-- {{if .Movie.IsFree}} ... {{end}} -->` yang membungkus sintaks Go template di bagian header sehingga memicu garis merah linter di VS Code / Cursor IDE. (2) Memindahkan variabel server-side JavaScript (`.Movie.ID`, `.TMDbID`, `isTV`, `trailerURL`) ke HTML data-attributes (`#movie-client-meta`), sehingga seluruh kode di dalam blok `<script>` kini murni 100% JavaScript standar ECMA tanpa tanda kurung kurawal ganda `{{...}}` yang memicu syntax error pada language server editor. (3) Memperbaiki event `onclick="switchServer(this)"` dan `onclick="switchEpisode(this)"` dengan pembacaan data-attribute untuk mengeliminasi peringatan `',' expected.` pada IDE. |
| 2026-09-16 | **Dokumentasi Eksekusi CLI di Linux Ubuntu (Docker & Host) di README.md & INSTRUKSI-SERVER.md:** Penambahan panduan komprehensif cara mengeksekusi seluruh varian perintah CLI terminal (Hollywood, Anime, Drama, Populate Streams, Downloads, Auto-Translate, Fix-Titles, Check-Links, Convert-Images) di server Linux Ubuntu, baik melalui container Docker `docker compose exec app ./cinembrot <flags>`, binary lokal `./cinembrot <flags>`, maupun `go run . <flags>`. |
| 2026-09-16 | **Konversi & Terjemahan Otomatis Judul Korea (Hangul) & CJK ke English Latin:** (1) Integrasi engine penerjemah cerdas (`translator.Translate`) pada scraper TMDb TV/Movie (`provider/tmdb/tmdb.go`) dan Jikan Anime (`provider/jikan/jikan.go`) sehingga jika data judul resmi English tidak disediakan oleh sumber, sistem otomatis menerjemahkan judul Hangul/Kanji ke huruf Latin Bahasa Inggris QWERTY. (2) Judul asli Hangul tetap tersimpan aman di `original_title` dan `alternative_titles` untuk pencarian dwibahasa. (3) Peningkatan utilitas `pipeline.FixNonLatinTitlesAndSynopses` (`.\cinembrot.exe -fix-titles`) dengan fallback penerjemah mandiri untuk menyisir dan mengonversi judul Korea lama di database. |
| 2026-09-16 | **Integrasi Cloudflare Turnstile CAPTCHA pada Login Admin (/admin/login):** (1) Modul baru `server/turnstile.go` untuk verifikasi token CAPTCHA sisi server via API Cloudflare Turnstile (`siteverify`). (2) Pemasangan widget Turnstile bertema gelap di formulir login CMS Admin (`server/views/admin_login.html`) dengan Site Key dan Secret Key resmi. (3) Penambahan pengontrolan Turnstile di `server/admin_handlers.go` (penolakan login otomatis jika verifikasi gagal atau belum dicentang). (4) Penyediaan kartu konfigurasi baru "Keamanan & Cloudflare Turnstile CAPTCHA" di CMS Admin Settings (`server/views/admin_settings.html` & `server/admin_settings_handlers.go`) untuk saklar ON/OFF dan pembaruan Site Key / Secret Key secara dinamis. |
| 2026-09-17 | **Peningkatan Maksimal SEO & Ads (ads.txt Dinamis, 40 Genre Sitemap, Schema Breadcrumb, Injeksi Head/Footer, Anti-AdBlock, & Buffer Interstitial Download):** (1) Route baru `GET /ads.txt` dan form editor `ads.txt` di CMS Admin Settings untuk verifikasi publisher jaringan iklan (Adsterra/DSP/AdSense) langsung dari MariaDB. (2) Ekspansi XML Sitemap (`/sitemap.xml`) yang mencakup seluruh 40 genre (`/genre/{slug}`) dan filter tahun rilis untuk menggenjot organic search traffic Google. (3) Rich Snippets Schema.org JSON-LD BreadcrumbList dan navigasi visual Breadcrumb di `detail.html`. (4) Penambahan tag `<link rel="alternate" hreflang="en">` dan `hreflang="id"` serta DNS preconnect/prefetch ke `image.tmdb.org` di `layout.html` untuk memangkas LCP Core Web Vitals. (5) Injeksi script kustom `custom_head_code` (`<head>`) dan `custom_footer_code` (`</body>`) via CMS Settings untuk Google Analytics GA4 / Histats. (6) Banner notifikasi Anti-AdBlocker ramah penonton dengan saklar ON/OFF di CMS Settings. (7) Modal Buffer Interstitial Smartlink Sponsor 3 detik saat penonton klik tombol download MP4/Subtitle dengan saklar ON/OFF di CMS Settings. |
| 2026-09-16 | **Penambahan Scheduler Harian Scraper Otomatis (05:00 & 17:00 WIB Setiap Hari):** (1) Pembuatan method baru `RunDailyCatchupCycle()` di `scheduler/scheduler.go` yang menjalankan seluruh varian scraper tahun berjalan (2026): Anime (Seasonal on-going & Top untuk cek episode/judul baru), Drama Asia (Korea, China, Jepang, Thailand untuk cek episode/judul baru), Hollywood (Now Playing bioskop & Popular untuk film baru), serta TMDb dan YTS. (2) Pembuatan pengawas jam mandiri `startDailyFixedScheduler()` di `scheduler/scheduler.go` yang menghitung mundur target jam 05:00 dan 17:00 WIB (`Asia/Jakarta`) secara presisi di background container. (3) Flag CLI baru `-daily-scrape` di `main.go` untuk eksekusi on-demand atau cron. (4) Handler CMS Admin `POST /admin/tools/run-daily-scrape` dan kartu status "Jadwal Harian Scraper Otomatis" di `server/views/admin_tools.html`. (5) Skrip host `~/docker/host/cinembrot-daily-scrape.sh` dan crontab user `taoen45` sebagai redundansi eksternal terjadwal. |
| 2026-09-17 | **Pemulihan Ruang Disk Root & Penyelesaian Error APT Upgrade:** (1) Pembersihan Docker BuildKit build cache (`docker builder prune -a -f`) yang membebaskan **78.79 GB** ruang disk di partisi root (`/dev/mapper/ubuntu--vg-ubuntu--lv`), mengembalikan sisa kapasitas bebas dari 0% menjadi 69 GB (26% used). (2) Pendaftaran SSH Public Key pengembang (`id_rsa.pub`) ke file `~/.ssh/authorized_keys` untuk user `root` dan `taoen45`, memungkinkan remote akses langsung tanpa hambatan dari Antigravity IDE / SSH CLI. (3) Pengalihan mirror repositori di `/etc/apt/sources.list.d/ubuntu.sources` dari mirror lokal yang error/HTTP 403 (`id.archive.ubuntu.com`) ke repositori resmi Ubuntu (`archive.ubuntu.com`). (4) Penyelesaian proses `apt update` dan `apt full-upgrade -y` serta pembersihan `apt autoremove --purge -y && apt clean` hingga 100% tuntas tanpa error. (5) Verifikasi keempat container Docker (`app`, `caddy`, `mysql`, `redis`) berjalan normal dan web server merespons HTTP 200. |
| 2026-09-17 | **Pembersihan Syntax Error & Linter Editor pada Template detail.html (isInterstitialEnabled):** Memindahkan evaluasi kondisi Go template `{{if and .EnableAds .EnableDownloadInterstitial}}` ke data attribute HTML `data-interstitial-enabled` pada elemen `#movie-client-meta`, dan membaca nilainya via `metaEl.dataset.interstitialEnabled`. Menghilangkan peringatan syntax error `Property assignment expected.` di VS Code / Cursor IDE sehingga blok `<script>` tetap 100% JavaScript valid. |
| 2026-09-17 | **Redesain Beranda Sinematik, Pembersihan Navigasi & Hardening Keamanan URL/Server:** (1) Pembersihan navigasi navbar: menghapus menu "Popular" dari navbar desktop dan sub-navbar mobile (`layout.html`). (2) Redesain total beranda (`home.html`): Hero Spotlight sinematik dengan mini-thumbnail navigation strip, baris Quick Access Category Pills (Hollywood, Anime, Drama, Rating 8.0+, Gratis), dan 5 rak horisontal (*carousel shelves* bergaya Netflix) untuk Box Office (#1-#12), Top Rated pilihan kritikus, Anime populer, dan Drama Asia populer dengan kontrol geser mulus. (3) Modul keamanan baru `server/security.go`: Global HTTP Security Headers (X-Frame-Options SAMEORIGIN, X-Content-Type-Options nosniff, X-XSS-Protection, Referrer-Policy, Permissions-Policy, HSTS) dan In-Memory Token Bucket Rate Limiter per IP (mencegah scraping bot, brute force login admin, dan DDoS). (4) Sanitasi dan validasi parameter URL: validasi regex slug anti-path traversal/SQLi, pembatasan integer pagination `?p=` (maks 500), sanitasi panjang query `?q=` (maks 100 char), validasi integer ID API, SameSite Lax session cookie, dan proteksi anti-open-redirect pada ganti bahasa/login. |
| 2026-09-17 | **Penyempurnaan Live Filter AJAX POST Seluruh Katalog, Penyelarasan Backend CMS & Pembersihan Navigasi Usang:** (1) Perbaikan syntax error template: memperbaiki duplikasi tag `{{end}}` di `drama_pendek.html` yang sempat menyebabkan panic pada `loadTemplates()` dan memutus koneksi server. (2) Smart Fallback Poster & TMDb CDN: handler `/uploads/` di `server/server.go` kini mengembalikan HTTP 404 (Not Found) jika file WebP lokal belum ada, sehingga browser langsung memicu atribut `onerror` dan memuat gambar resmi dari TMDb CDN (`image.tmdb.org`) alih-alih placeholder "Preview Tidak Tersedia". (3) AJAX POST Live Filter Publik: implementasi controller AJAX di `anime.html` dan `hollywood.html` dengan event listener auto-trigger `change` pada dropdown (tahun, genre, negara, urutan), sinkronisasi address bar via `history.pushState`, status Live Filter badge, dan loading spinner tanpa reload halaman. (4) Penyelarasan Backend CMS Admin: default sorting di `HandleAdminMovies`, `HandleAdminHollywood`, `HandleAdminAnime`, `HandleAdminDramaPendek`, dan `HandleAdminDashboard` diubah dari `id desc` menjadi rilis terbaru (`year desc, release_date desc, id desc`), serta penambahan atribut `data-fallback` dan `onerror` pada tag poster di `admin_movies.html` dan `admin_dashboard.html`. (5) Pembersihan Navigasi Backend: menghapus menu usang "Sumber Website" (`/admin/sources`) dari sidebar CMS Admin di `server/views/admin_layout.html` agar panel kontrol lebih ringkas dan fokus pada "Alat & Scraper" (`/admin/tools`). |







