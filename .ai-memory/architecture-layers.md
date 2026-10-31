# Arsitektur 3-Layer untuk Efisiensi Token AI Coding Agent (Cinembrot)

Dokumen ini mendefinisikan arsitektur 3-Layer yang disesuaikan khusus untuk project **CINEMBROT (Golang Native + MariaDB + Redis + Caddy)** agar menghemat token hingga 75%–85%, mencegah pembengkakan context window, dan mengeliminasi halusinasi.

---

## 1. Diagram Arsitektur

```
                    ┌────────────────────────────────────────────────────────┐
                    │            Cursor IDE / Antigravity AI Agent           │
                    │                    (Pusat Kendali)                     │
                    └───────────────────────────┬────────────────────────────┘
                                                │
         ┌──────────────────────────────────────┴──────────────────────────────────────┐
         ▼                                                                             ▼
┌─────────────────────────────────┐                           ┌─────────────────────────────────┐
│ Layer 1 — Context Management    │                           │ Layer 2 — Execution             │
│ • AI Memory (.ai-memory/)       │                           │ • MCP Server (cinembrot)        │
│   - Aturan bisnis & streaming   │                           │   - Safe Read-Only DB           │
│   - SOP Zero Data Loss DB       │                           │   - Search Project RAG (.md)    │
│ • Modular Rules (.cursor/rules/)│                           │   - Targeted Log Diagnosis      │
│ • RAG Search (search_project)   │                           │ • Terminal (Go CLI, Git, curl)  │
│   - Chunked search tanpa dump   │                           │   - Non-destructive execution   │
└────────────────┬────────────────┘                           └────────────────┬────────────────┘
                 │                                                             │
                 └──────────────────────────────┬──────────────────────────────┘
                                                │
                                                ▼
                               ┌─────────────────────────────────┐
                               │ Layer 3 — Validation            │
                               │ • Playwright (Browser E2E Test) │
                               │ • Go Test (Unit & Handlers)     │
                               │ • Compact Test Runner di MCP    │
                               │   (Passed/Failed summary)       │
                               └────────────────┬────────────────┘
                                                │
                                                ▼
                               ┌─────────────────────────────────┐
                               │ Target: Cinembrot Application   │
                               │ (Golang Engine, Caddy, MariaDB) │
                               └─────────────────────────────────┘
```

---

## 2. Rincian Layer

### Layer 1 — Context Management (Hemat Token Input)
- **Tujuan**: Mencegah model membaca seluruh file code / log yang masif ke dalam context prompt.
- **Komponen**:
  1. `.ai-memory/`: Dokumen pengetahuan terpecah per domain (`business-domain-summary.md`, `database-safety-policy.md`, `token-optimization-guide.md`).
  2. `search_project_rag` (Tool MCP): Melakukan semantic/keyword chunking pada dokumentasi `.ai-memory/` dan panduan server, hanya mengembalikan potongan teks relevan (~200 token) alih-alih seluruh isi file (~15.000 token).

### Layer 2 — Execution (Terkontrol & Aman)
- **Tujuan**: Eksekusi command dan query dengan pagar pengaman tanpa risiko kehilangan data.
- **Komponen**:
  1. `inspect_db_schema_safe`: Inspeksi skema MariaDB secara 100% Read-Only.
  2. `diagnose_server_logs`: Diagnosa log Docker Go, Caddy, Scraper, Cloudflared secara presisi dengan filter level dan keyword regex.
  3. **Proteksi Database**: Segala bentuk modifikasi data (INSERT, UPDATE, DELETE, ALTER, DROP) **WAJIB** meminta izin/konfirmasi tertulis kepada user terlebih dahulu.

### Layer 3 — Validation (Verifikasi Cepat & Ringkas)
- **Tujuan**: Memvalidasi fungsionalitas aplikasi tanpa memenuhi token dengan log output terminal ribuan baris.
- **Komponen**:
  1. `run_tests` (Tool MCP): Menjalankan `go test ./...` atau `playwright test` dan hanya mengembalikan kesimpulan ringkas: status `passed/failed`, jumlah test, dan cuplikan 5 baris error teratas jika gagal.
  2. Playwright E2E (`tests/e2e/smoke.spec.js`): Smoke test headless untuk menguji homepage, katalog anime/drama/hollywood, dan pemutar player streaming.
