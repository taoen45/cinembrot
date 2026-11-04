---
name: ad-revenue-optimizer
description: Penempatan unit iklan strategis, monetisasi Adsterra & DSP, ads.txt, perlindungan Core Web Vitals, dan Anti-AdBlock ramah pengguna.
---

# Ad Revenue Optimizer Skill

## Prinsip Monetisasi Berkelanjutan
1. **Perlindungan Core Web Vitals (LCP & CLS)**:
   - Sediakan kontainer banner dengan dimensi statis (misal `min-h-[250px]` atau `min-h-[90px]`) agar pemuatan iklan tidak memicu Cumulative Layout Shift (CLS).
   - Muat script iklan (Popunder/Social Bar) secara asinkron (`async`/`defer`) setelah event `DOMContentLoaded`.
2. **Manajemen ads.txt Dinamis**:
   - Deklarasikan baris publisher resmi di `/ads.txt` dan verifikasi langsung via database setting CMS.
3. **Smartlink & Buffer Interstitial**:
   - Berikan modal jeda hitung mundur (3 detik) sebelum download untuk memaksimalkan CTR Smartlink tanpa merusak retensi penonton.
4. **Anti-AdBlock Ramah Pengunjung**:
   - Deteksi adblocker secara sopan tanpa memblokir akses total penonton; tampilkan notifikasi edukatif untuk mendukung keberlangsungan server gratis.
