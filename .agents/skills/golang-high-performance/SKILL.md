---
name: golang-high-performance
description: Optimalisasi performa Golang, goroutines, channel concurrency, GORM connection pooling, dan pencegahan memory leak.
---

# Golang High Performance Skill

## Standar Performa Mesin Go Cinembrot
1. **Concurrency & Resource Management**:
   - Selalu batasi jumlah konkurensi goroutine menggunakan buffered channel semaphore atau `sync.WaitGroup` agar tidak terjadi lonjakan konsumsi CPU/RAM.
   - Pastikan setiap `http.Response.Body` selalu ditutup dengan `defer resp.Body.Close()` dan dikonsumsi penuh (`io.Copy(io.Discard, resp.Body)`) agar koneksi TCP dapat digunakan kembali (Keep-Alive).
2. **Database Pooling & GORM**:
   - Konfigurasi connection pool secara presisi: `SetMaxOpenConns(50)`, `SetMaxIdleConns(10)`, dan `SetConnMaxLifetime(1 * time.Hour)`.
   - Hindari query N+1 pada list katalog; manfaatkan `Preload` selektif dengan field spesifik (`Select`).
3. **HTTP Server & Zero-Allocation**:
   - Gunakan `strings.Builder` atau `byte buffer` untuk perakitan teks dinamis.
   - Manfaatkan context timeout (`http.Client{Timeout: ...}`) pada setiap integrasi client eksternal.

