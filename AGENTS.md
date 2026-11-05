# AGENTS.md — Server taoen45

Sebelum mengubah, memperbaiki, atau menambah apa pun di server ini:

1. **Baca dulu** `/home/taoen45/INSTRUKSI-SERVER.md` (sumber kebenaran: spec, tujuan, aplikasi, jadwal, jaringan).
2. **Ikuti** aturan Cursor `/home/taoen45/.cursor/rules/server-instruksi.mdc`.
3. Jika tidak jelas, **tanya user**. Jangan mengarang.
4. **Database Wajib Izin**: Urusan database harus ekstra hati-hati. Operasi delete, restore, edit, atau modifikasi skema/data **WAJIB meminta izin (permission)** dari user terlebih dahulu.
5. **Bahasa Indonesia**: Selalu menggunakan Bahasa Indonesia dalam percakapan, instruksi, dan dokumentasi.
6. **Anti-Scraping / Vorta API Streaming Safety Rule (WAJIB DIPATUHI OLEH AGENT SAAT MEMBUAT BOT)**:
   - **TIDAK BOLEH** melakukan bulk-scraping, mass-harvesting, atau mendownload seluruh database katalog sekaligus (dilarang keras oleh penyedia API / STRICT WARNING: "Once detected pulling data in bulk, your API key is PERMANENTLY BLOCKED. No warning, no negotiation, no refund, and no second chance").
   - **On-Demand & Jeda Acak (Random Jitter/Delay)**: Bot hanya boleh memanggil data secara on-demand / bertahap (fetch only what you need, when you need it). Setiap pemanggilan wajib menerapkan delay acak (misal 5 hingga 15 detik), rotasi user-agent wajar, dan bergantian platform (misal MeloShort lalu ganti ShortMax lalu jeda lama).
   - Jangan menjalankan looping masif atau konkurensi paralel tinggi ke endpoint streaming/katalog Vorta API.
7. Setelah menambah fitur/aplikasi/timer/port: **update** `INSTRUKSI-SERVER.md` (daftar + Riwayat perubahan).
8. Setiap perubahan pada file instruksi/agent (`AGENTS.md`, aturan Cursor, `INSTRUKSI-SERVER.md`) atau sistem **wajib dicatat** di `INSTRUKSI-SERVER.md` bagian **Riwayat perubahan**.

Stack: **CINEMBROT (Golang)** di `~/docker/html` + MariaDB + Redis + Caddy (proxy ke `:8080`) di `~/docker`. **Golang tidak butuh PHP.** Jellyfin hanya cadangan (profile, tidak aktif).
Publik: Cloudflare Tunnel:
- Domain Utama: `cinembrot.web.id` → `http://localhost:80` → Caddy → Go `:8080`.
- Domain Informasi: `cinembrot.my.id` → `http://localhost:81` → Caddy → Go `:8081` (Landing page info resmi gaya filmapik.info).
`DB_HOST` app = `127.0.0.1` (MariaDB host network).
Pemeliharaan: update apt **03:40 WIB**, reboot **04:00 WIB**. Timezone **Asia/Jakarta**.

