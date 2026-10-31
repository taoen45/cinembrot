# Panduan Optimasi Context Token AI Coding Agent

Dokumen ini adalah acuan kerja agar AI Agent tidak memboroskan token context window pada proyek Cinembrot.

---

## 1. Hindari Dump File Utuh
- Jangan menjalankan perintah `cat` atau membaca file ribuan baris sekaligus seperti `INSTRUKSI-SERVER.md` (60.000+ karakter) atau log server secara mentah jika hanya mencari informasi spesifik.
- Gunakan tool MCP `search_project_rag` untuk mendapatkan potongan bab yang dicari.

## 2. Diagnosa Log Hemat Token
- Gunakan tool MCP `diagnose_server_logs` dengan parameter `filterKeyword` (misal `error`, `panic`, `timeout`) dan batasi baris `lines: 20` s/d `50`.
- Jangan pernah menjalankan `docker logs` tanpa `--tail`.

## 3. Eksekusi Test Ringkas
- Gunakan tool MCP `run_tests` dengan suite `go` atau `playwright`.
- Tool ini memfilter output terminal ratusan baris dan hanya memancarkan ringkasan:
  - Jumlah test passed / failed.
  - Baris error spesifik (jika ada kegagalan).

## 4. Pelaporan ke User
- Hindari menyalin ulang isi dokumen/skrip panjang di dalam chat bubble.
- Berikan ringkasan poin keputusan dan arahkan ke file/artifact terkait.

