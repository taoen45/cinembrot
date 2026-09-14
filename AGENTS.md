# AGENTS.md — Server taoen45

Sebelum mengubah, memperbaiki, atau menambah apa pun di server ini:

1. **Baca dulu** `/home/taoen45/INSTRUKSI-SERVER.md` (sumber kebenaran: spec, tujuan, aplikasi, jadwal, jaringan).
2. **Ikuti** aturan Cursor `/home/taoen45/.cursor/rules/server-instruksi.mdc`.
3. Jika tidak jelas, **tanya user**. Jangan mengarang.
4. **Database Wajib Izin**: Urusan database harus ekstra hati-hati. Operasi delete, restore, edit, atau modifikasi skema/data **WAJIB meminta izin (permission)** dari user terlebih dahulu.
5. **Bahasa Indonesia**: Selalu menggunakan Bahasa Indonesia dalam percakapan, instruksi, dan dokumentasi.
6. Setelah menambah fitur/aplikasi/timer/port: **update** `INSTRUKSI-SERVER.md` (daftar + Riwayat perubahan).
7. Setiap perubahan pada file instruksi/agent (`AGENTS.md`, aturan Cursor, `INSTRUKSI-SERVER.md`) atau sistem **wajib dicatat** di `INSTRUKSI-SERVER.md` bagian **Riwayat perubahan**.

Stack: **CINEMBROT (Golang)** di `~/docker/html` + MariaDB + Redis + Caddy (proxy ke `:8080`) di `~/docker`. **Golang tidak butuh PHP.** Jellyfin hanya cadangan (profile, tidak aktif).
Publik: Cloudflare Tunnel → `http://localhost:80` → Caddy → Go `:8080`. Domain uji `test.cinembrot.my.id`; domain utama `cinembrot.my.id` (Path kosong di Zero Trust).
`DB_HOST` app = `127.0.0.1` (MariaDB host network).
Pemeliharaan: update apt **03:40 WIB**, reboot **04:00 WIB**. Timezone **Asia/Jakarta**.

