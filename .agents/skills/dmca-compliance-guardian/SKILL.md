---
name: dmca-compliance-guardian
description: Penanganan komplain DMCA, otomatisasi HTTP 404/410, isolasi konten terlarang, sanitasi sitemap, dan draf tanggapan resmi Cloudflare.
---

# DMCA Compliance Guardian Skill

## Prosedur Standar Operasional (SOP) DMCA
1. **Zero Data Loss Policy**:
   - Konten yang dilaporkan **TIDAK PERLU DIHAPUS PERMANEN** dari database. Cukup ubah kolom status menjadi `takedown` atau `unpublish`.
2. **Kepatuhan HTTP 404 / 410 Publik**:
   - Handler detail film (`HandleMovieDetail`) wajib mengembalikan respon `http.NotFound(w, r)` (HTTP 404) untuk status `takedown`/`unpublish`/`draft`.
   - Nonaktifkan stream link terkait di MariaDB (`is_valid = 0, status = 'DEAD'`).
3. **Sanitasi Mesin Pencari & Sitemap**:
   - Pengecualian otomatis slug berstatus takedown dari `sitemap.xml` agar perayap Google segera meng-unindex URL tersebut.
4. **Respon Resmi Pelapor**:
   - Buatkan balasan sopan dan profesional ke tim abuse (misal: `abusereply@cloudflare.com`) dengan menyertakan Report ID dan bukti verifikasi respon HTTP 404.

