# Integrasi Vorta API untuk Drama Pendek (Cinembrot)

Dokumen ini mendokumentasikan integrasi resmi Vorta Tech REST API (`https://api.vorta.tech`) di Cinembrot untuk katalog Drama Pendek Asia.

---

## 1. Spesifikasi Kredensial & Batasan
- **Base URL**: `https://api.vorta.tech`
- **Tipe Autentikasi**: Signature HMAC-SHA256
  - Header `X-Timestamp`: Unix epoch milliseconds
  - Header `X-Signature`: `HMAC-SHA256(SecretKey, "GET:" + path + ":" + timestamp)`
- **Plan**: Trial (600 requests / hari, reset pukul 00:00 UTC / 07:00 WIB)
- **Platform Aktif**: `meloshort`, `shortmax`, `mydrama`
- **Fitur Utama**: Mendukung video streaming HLS/M3U8 langsung (`has_video_access: true`).

---

## 2. Aturan Ketat (Anti Mass-Harvesting / Zero Ban Policy)
> ⚠️ **PERINGATAN KERAS DARI DOKUMENTASI VORTA**:
> *"This API is for streaming, not for harvesting data. Bulk-scraping is strictly forbidden. Once detected pulling data in bulk, API key is PERMANENTLY BLOCKED."*

### SOP Penggunaan di Cinembrot:
1. **On-Demand Fetching (Bukan Bulk Download)**:
   - DILARANG mengunduh seluruh 60 episode sekaligus dalam satu waktu.
   - Panggilan ke `/api/v2/video` hanya dilakukan saat pengunjung membuka detail episode terkait (just-in-time / on-demand playback).
2. **Caching Sederhana**:
   - Cache URL stream M3U8 jangka pendek pada Redis / in-memory cache untuk mencegah request berulang pada episode yang sama dalam rentang menit.
3. **Mata Rantai Endpoint**:
   - `GET /api/v2/home?category_p=meloshort&lang=id` -> Mendapatkan katalog beranda.
   - `GET /api/v2/detail?category_p=meloshort&id={drama_id}&lang=id` -> Mendapatkan metadata judul & daftar chapter.
   - `GET /api/v2/video?category_p=meloshort&id={drama_id}&chapterId={chapter_id}&lang=id` -> Mendapatkan URL streaming M3U8 (`resolution: 1280x720`).

