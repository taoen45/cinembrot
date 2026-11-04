---
name: hls-stream-guard
description: Manajemen streaming video HLS (M3U8), reverse proxy video, bypass hotlink CDN, penanganan DRM token, dan CORS headers.
---

# HLS Stream Guard Skill

## Standar Streaming Video HLS & M3U8
1. **CORS & Video Player Integration**:
   - Pastikan header streaming mengizinkan CORS: `Access-Control-Allow-Origin: *` dan `Access-Control-Allow-Methods: GET, OPTIONS, HEAD`.
   - Gunakan Video.js atau Hls.js ringan untuk rendering playlist M3U8 dengan graceful degradation ke MP4.
2. **Reverse Proxy & Referer Protection**:
   - Jika CDN sumber menerapkan proteksi hotlinking/referer, arahkan segmen melalui reverse proxy Go internal (`/api/stream/proxy?url=...`) dengan menyertakan header `Referer` dan `User-Agent` yang sah.
3. **AES-128 & Stream Token Neutralization**:
   - Tangani key URI pada tag `#EXT-X-KEY:METHOD=AES-128,URI="..."` agar dapat diakses oleh player browser tanpa error certificate atau blocked domain.
