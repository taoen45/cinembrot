# Aturan & Kebijakan Keamanan Database (Zero Data Loss)

> **ATURAN MUTLAK SERVER TAOEN45 & CINEMBROT**:
> Database Wajib Izin! Operasi modifikasi data atau skema apapun WAJIB konfirmasi kepada user terlebih dahulu.

---

## 1. Operasi yang DILARANG Tanpa Izin Tertulis User:
1. `DELETE FROM ...` atau `TRUNCATE TABLE ...`
2. `UPDATE ...` secara massal maupun per record.
3. `INSERT INTO ...` langsung via query tool tanpa konfirmasi.
4. `DROP TABLE`, `DROP DATABASE`, `ALTER TABLE ... DROP COLUMN`.
5. Restore database atau import SQL mentah yang berpotensi menimpa baris yang sudah ada.

---

## 2. Praktik Aman:
- Gunakan tool MCP `inspect_db_schema_safe` untuk membaca struktur tabel dan statistik secara **READ-ONLY**.
- Jika user meminta perbaikan data, siapkan rencana query terlebih dahulu, jelaskan dampaknya, dan tunggu jawaban user sebelum mengeksekusi.
- Pastikan koneksi MariaDB mengarah ke host `55.0.0.157:3306` (atau `127.0.0.1` saat di dalam server host) dengan database `cinembrot`.

