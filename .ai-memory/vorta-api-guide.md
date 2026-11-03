# Vorta API & Stealth Drama Scraper Guide (Cinembrot)

Dokumen panduan arsitektur integrasi Vorta Tech REST API (`https://api.vorta.tech`) dan modul scraper sopan anti-ban di Cinembrot.

---

## 1. Kredensial & Batasan Akun
- **Base URL**: `https://api.vorta.tech`
- **Secret Key**: `b15bc8fff70cc4acdb67f6e8a274c065c1c8d579c2297782ac5bb017d1871ae2` (Disimpan aman di `.env` sebagai `VORTA_API_SECRET`)
- **Autentikasi**: HMAC-SHA256 signature (`X-Timestamp` unix epoch ms, `X-Signature: hex(hmac_sha256(secret, "GET:" + path + ":" + ts))`)
- **Plan**: Trial (Batas 600 requests / hari, kuota direset tiap 00:00 UTC / 07:00 WIB)
- **Platform Pool**: `meloshort`, `shortmax`, `mydrama`
- **Tipe Video**: HLS Streaming Playlist (`.m3u8`) 720p / MP4 dengan validasi auth key

---

## 2. Sample Data Tersimpan di MariaDB (ID: 8111)
- **Judul**: *Raja Wall Street Melindungi Putrinya*
- **Slug**: `raja-wall-street-melindungi-putrinya-2026`
- **URL Detail**: `https://cinembrot.web.id/movie/raja-wall-street-melindungi-putrinya-2026`
- **Stream Link ID**: 31441
- **Server Name**: `Server 1 - Meloshort HD (M3U8)`
- **Format**: `720p HLS`

---

## 3. Kebijakan Anti-Harvesting / Anti-Ban Vorta
Penyedia API menerapkan aturan ketat:
> *"This API is for streaming, not for harvesting data. Bulk-scraping API data is prohibited... Once detected pulling data in bulk, your API key is PERMANENTLY BLOCKED."*

### Pilar Pertahanan Cinembrot:
1. **Human-like Jitter Delay**: Setiap request disisipi jeda acak manusiawi (misal 5-15 detik) menggunakan generator acak `time.Duration(min + rand(max - min))`.
2. **Platform Shuffling**: Pengambilan feed tidak monoton dari satu sumber, melainkan diacak antara `meloshort`, `shortmax`, dan `mydrama`.
3. **Random Item Traversal**: Urutan item dalam feed beranda diacak sebelum diproses, menghindari pola indeks statis (0, 1, 2, ...).
4. **On-Demand Streaming (Zero Bulk Download)**: Hanya mengambil sampel 1 episode saat mencatat judul. Episode-episode lanjutan diambil on-demand saat dibuka oleh penonton.
5. **Duplicate Skip**: Jika judul atau slug sudah ada di MariaDB, request detail di-skip untuk menghemat kuota harian.

---

## 4. Perintah Operasional CLI
```powershell
# Windows PowerShell (Local Test):
.\cinembrot.exe -scrape-short-drama -short-limit 1 -short-min-delay 5 -short-max-delay 15

# Linux Ubuntu (Docker di Server taoen45):
docker compose exec app ./cinembrot -scrape-short-drama -short-limit 1
```
