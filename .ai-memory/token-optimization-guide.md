# Panduan Efisiensi Token Maksimal & Prinsip Ponytail (Cinembrot)

Panduan ini wajib dipatuhi oleh Cursor IDE, Antigravity, GitHub Copilot, Claude Code, dan AI agent lainnya saat bekerja di repositori **CINEMBROT (Golang Native)**.

---

## 1. Golden Rules Efisiensi Token
1. **JANGAN Membaca File Utuh Jika Cukup Membaca Chunk:**
   - Gunakan pembacaan baris tertarget (`StartLine` & `EndLine`) saat membaca file berukuran besar (`main.go`, `handlers.go`, `INSTRUKSI-SERVER.md`).
   - Jangan dump seluruh file Go atau template HTML jika hanya butuh 1 fungsi atau 1 blok elemen.

2. **Gunakan MCP / RAG Daripada Raw Grep Menyeluruh:**
   - Gunakan tool MCP `search_project_rag` untuk mendapatkan potongan pengetahuan bisnis, arsitektur, atau panduan server tanpa memuat file `.md` secara utuh.
   - Gunakan tool MCP `inspect_db_schema_safe` untuk memeriksa tabel dan struktur kolom MariaDB secara read-only tanpa query SQL besar.

3. **Gunakan Ringkasan Log, Bukan File Log Mentah:**
   - DILARANG melakukan dump log server secara mentah ke context prompt.
   - Gunakan tool MCP `diagnose_server_logs` dengan target spesifik (`app`, `caddy`, `scraper`, `maintenance`) dan filter keyword regex.

4. **Validasi Mandiri via Layer 3 (Go Tests & Playwright E2E):**
   - Sebelum melaporkan selesai ke user, validasi perubahan kode dengan tool MCP `run_tests` (`suite: "go"` atau `suite: "playwright"`).
   - Tool `run_tests` mengompresi ribuan baris output terminal menjadi ringkasan JSON yang sangat ringkas (< 50 token).

5. **Jawaban Ringkas & Langsung ke Inti (Compact Response):**
   - Hindari basa-basi dan pengantar teori yang panjang.
   - Tampilkan potongan diff terfokus, bukan mencetak ulang seluruh file yang dimodifikasi.

---

## 2. Prinsip Ponytail (Lazy Senior Dev / Minimalist Code)
*Filosofi utama: "The best code is the code you never wrote. Kode paling hemat token dan paling stabil adalah kode paling ringkas yang tepat sasaran."*

### A. The Decision Ladder (Hirarki Pengambilan Keputusan)
Sebelum menulis baris kode baru, AI agent WAJIB mengevaluasi tingkatan berikut dari atas ke bawah:
1. **Tingkat 1 — No New Code (Cukup Konfigurasi / Env / Existing Feature):**
   - Apakah sudah ada helper, fungsi provider, model GORM, atau konfigurasi bawaan Cinembrot yang menyediakannya?
   - Manfaatkan fitur yang sudah ada tanpa menulis ulang logika serupa.
2. **Tingkat 2 — Minimal Surgical Edit (1–5 Baris):**
   - Selesaikan masalah dengan mengubah seminimal mungkin baris pada file yang sudah ada.
   - DILARANG melakukan refactoring acak pada kode di sekitarnya yang sedang bekerja normal (*don't touch running code*).
3. **Tingkat 3 — DILARANG Over-Engineering (YAGNI - You Aren't Gonna Need It):**
   - Dilarang membuat package/struct baru, interface bertingkat, middleware wrapper baru, atau file helper baru jika kebutuhannya hanya untuk 1 handler atau 1 endpoint.
   - Dilarang menambahkan dependency Go external baru (`go get`) jika bisa diselesaikan dengan Go standard library bawaan.

### B. Anti-Bloat Output Guidelines (Hemat Token Output)
- **Tampilkan Diff Tertarget:** Jangan mencetak ulang seluruh file jika hanya mengubah 2 baris kondisi `if`.
- **Hentikan Diskusi Teori:** Langsung ke akar masalah dan aksi nyata. Jangan menulis esai pengantar tentang konsep umum pemrograman.
- **Transparansi Singkat:** Jika ada asumsi atau bagian yang sengaja tidak disentuh demi keamanan sistem atau database, sebutkan secara singkat (1 kalimat).
