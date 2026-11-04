---
name: whatsapp-funnel-architect
description: Perancangan funnel transaksi WhatsApp, auto-routing sales, pesan pre-filled cerdas, dan generator tautan API WhatsApp.
---

# WhatsApp Funnel Architect Skill

## Strategi WhatsApp Funnel & Click-to-Chat
1. **Link Builder & Pre-filled Messages**:
   - Format standar: `https://wa.me/{number}?text={encoded_message}`.
   - Pesan pre-filled memuat konteks: Nama produk/film, kode ID, dan kebutuhan pengunjung (misal: "Halo Admin, saya ingin request film: [Judul]").
2. **Auto-Routing & Load Balancing**:
   - Distribusi lead/pesan ke beberapa nomor customer service/admin secara bergantian (round-robin rotasi).
3. **Tracking & Analytics**:
   - Injeksi event tracking (Google Analytics / Pixel) saat tombol WhatsApp diklik untuk mengukur rasio konversi.

