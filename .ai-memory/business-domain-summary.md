# Ringkasan Domain Bisnis & Fitur Cinembrot

Cinembrot adalah platform movie streaming & subtitle engine berbasis **Golang native** (tanpa PHP) yang didesain untuk performa tinggi, hemat resource, dan teroptimasi SEO.

---

## 1. Stack & Infrastruktur
- **Backend**: Golang 1.22+ Native (Gin/Standard HTTP Router, GORM ORM).
- **Database**: MariaDB (host network server `taoen45` di WireGuard `55.0.0.157:3306`, DB: `cinembrot`).
- **Cache & Session**: Redis 7 di `127.0.0.1:6379`.
- **Web Server / Reverse Proxy**: Caddy server (port `:80` & `:81` proxy ke Go `:8080` & `:8081`).
- **Publik via Cloudflare Tunnel**:
  - `cinembrot.web.id` -> Streaming Hub Utama (`:8080`).
  - `cinembrot.my.id` -> Landing Page Informasi Resmi (`:8081`).

---

## 2. Fitur-Fitur Kunci
1. **Multi-Server Streaming Player**:
   - 7 Server Embed: VidSrc, AutoEmbed, 2Embed, VidLink, SuperEmbed, MultiEmbed, NontonGo, plus YouTube Trailer.
   - Dukungan serial berepisode (Anime & Asian Drama) dengan episode selector client-side.
2. **Katalog Terpadu**:
   - Hollywood / Box Office (`/hollywood`).
   - Anime (`/anime`) via MyAnimeList / Jikan API.
   - Drama Pendek & Asian Drama (`/drama-pendek`) via TMDb TV.
3. **Optimasi Gambar WebP**:
   - Semua poster dan backdrop otomatis dikonversi ke WebP lokal (Full & Thumbnail) hemat storage hingga 90%.
4. **Dwibahasa (ID / EN) & Auto-Translate**:
   - Bahasa default Indonesia. Sinopsis otomatis diterjemahkan via engine translasi ke ID dan versi EN disimpan terpisah.
5. **Monetisasi & SafelinkU REST API**:
   - Masa tenang iklan 5 menit (Ad cooldown) agar pengunjung nyaman.
   - Tombol download MP4/Subtitle otomatis dipendekkan ke tautan SafelinkU berbayar.
   - Buffer intersitial 3 detik untuk sponsor Adsterra.
