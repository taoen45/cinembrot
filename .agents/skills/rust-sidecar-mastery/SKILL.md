---
name: rust-sidecar-mastery
description: Komputasi super-cepat sidecar Rust, high concurrency, asynchronous runtime (Tokio), dan memory-safe rate limiting.
---

# Rust Sidecar Mastery Skill

## Arsitektur Sidecar & High Concurrency
1. **Pola Sidecar**:
   - Menjalankan tugas komputasi intensif (hashing, stream decryption, video remuxing, token bucket rate limiter) melalui modul/binary Rust terpisah (`crates/media-worker`).
2. **Koneksi IPC / CLI Bridge**:
   - Komunikasi ringan via stdin/stdout JSON streaming atau CGO/FFI fallback bridge di Go (`torrentmgr/rust_bridge.go`).
3. **Safety & Zero-Cost Abstraction**:
   - Memory safety tanpa garbage collection overhead.
   - Pemanfaatan `rayon` untuk paralelisme CPU-bound dan `tokio` untuk concurrent I/O.
