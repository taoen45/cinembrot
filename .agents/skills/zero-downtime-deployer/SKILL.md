---
name: zero-downtime-deployer
description: Deployment aman ke server produksi (Docker/Caddy), proteksi data MariaDB (Zero Data Loss), rollback plan, dan auto-sync remote.
---

# Zero-Downtime Deployer Skill

## Standar Deployment Aman Produksi
1. **Aturan Database Wajib Izin**:
   - Seluruh operasi database DDL/DML modifikasi WAJIB meminta izin dari pengguna terlebih dahulu.
   - Tidak ada operasi `DROP TABLE` atau migrasi destruktif langsung di produksi tanpa cadangan data.
2. **Workflow Deployment Standar**:
   - Commit lokal $\rightarrow$ Push ke remote fork GitHub $\rightarrow$ Pull ke remote host server `Ubuntu-FDME` (`/home/taoen45/docker/html`).
   - Reload layanan via `docker compose restart app` atau gracefully reload via Caddy.
3. **Healthcheck & Verifikasi**:
   - Verifikasi respon HTTP 200/404 menggunakan `curl -s -o /dev/null -w "%{http_code}\n"` segera setelah restart container.
   - Pengecekan container logs (`docker logs --tail 50`) untuk memastikan tidak ada panic atau fatal error.
4. **Pencatatan Riwayat**:
   - Setiap perubahan sistem dan port wajib dicatat di `/home/taoen45/INSTRUKSI-SERVER.md` bagian Riwayat Perubahan.
