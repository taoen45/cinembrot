# Panduan Desain & Standar Visual Cinembrot (Anti-AI UI)

Dokumen ini adalah acuan kerja agar antarmuka web Cinembrot tidak memiliki ciri-ciri generik buatan AI (*anti-slop*).

---

## 1. Aturan Penulisan & Karakter Teks (Anti-AI Cliché)
- **Eliminasi Tanda Em-Dash (`—`)**: Tanda em-dash adalah indikator kuat teks yang dihasilkan oleh model AI.
  - Ganti em-dash dengan tanda titik dua (`:`), tanda kurung `()`, titik kalimat, atau koma.
- **Copywriting Manusiawi**:
  - Hindari jargon klise AI (*seamless*, *revolutionary*, *empower*, *next-gen*, *unleash*).
  - Tulis kalimat praktis, ringkas, dan langsung pada intinya sesuai kebutuhan penonton Indonesia.

---

## 2. Standar Visual Frontend Golang (Tailwind + CSS)
- **Bukan Template Generik**: Hindari 3 kartu identik bertabur gradien neon ungu-biru.
- **Hierarki Sinematik**: Gunakan rak geser horisontal (Netflix-style carousel), kartu dengan rasio 2:3 poster film, badge kualitas (HD, 4K, CAM), dan rating bintang emas yang jelas.
- **State Interaktif**: Setiap tombol pemilih server streaming (Server 1 s/d Server 7) dan episode wajib memiliki state hover, active, focus, dan feedback visual saat diklik.
- **Palet Warna**: Mengutamakan neutral dark (Zinc/Slate) dengan aksen warna primer yang fungsional (amber/merah bioskop).

