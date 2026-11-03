# Aturan Desain Frontend & UI/UX Cinembrot (Anti-AI Cliché)

Panduan ini mendefinisikan standar visual dan copywriting untuk ekosistem **CINEMBROT (Golang Native)** agar antarmuka web terasa alami, fungsional, dan bebas dari kesan generic buatan AI ("AI-looking").

---

## 1. Aturan Tipografi & Copywriting (Anti-AI Cliché)

### A. Larangan Tanda Baca Ciri Khas AI
- **DILARANG menggunakan tanda em-dash (`—` / `\u2014`) atau en-dash berlebihan (`–`).**
  - AI sangat sering memakai em-dash untuk memisahkan klausa panjang.
  - **Pengganti Terbaik**:
    - Gunakan tanda titik dua (`:`) untuk sub-judul atau label domain (contoh: `Unit Iklan: Domain Utama`).
    - Gunakan tanda kurung `(...)` untuk keterangan tambahan.
    - Gunakan kalimat terpisah dengan tanda titik (`.`) atau koma (`,`).
    - Gunakan tanda minus standar (`-`) jika memang konteks rentang angka (misal: `2024-2026`).

### B. Larangan Copywriting Klise AI
- DILARANG menggunakan kata-kata marketing generik AI seperti:
  - *"Revolutionize"*, *"Seamless"*, *"Empower your streaming"*, *"Next-gen"*, *"Ultimate experience"*, *"Dive into"*.
- **Prinsip Teks Cinembrot**:
  - Tulis teks UI yang to-the-point, informatif, dan relevan dengan penonton film/anime di Indonesia.
  - Contoh: Alih-alih *"Nikmati pengalaman streaming seamless next-gen"*, gunakan *"Nonton streaming film dan anime dengan server cepat dan subtitle Indonesia."*

---

## 2. Aturan Visual & Komponen UI (Golang HTML Template + Tailwind)

### A. Larangan Efek Visual Klise AI
- DILARANG menggunakan gradien ungu-biru neon (*purple/blue dark mesh glow*).
- DILARANG menggunakan efek glassmorphism murahan yang transparan buram berlebihan dan merusak kontras baca.
- Hindari layout 3 kolom identik yang monoton. Gunakan layout sinematik: Bento grid variatif, rak horisontal (*carousel shelves*), atau spotlight preview.

### B. Palet Warna & Spatial Rhythm
- **Warna Dasar**: Slate / Zinc netral dengan kontras gelap yang nyaman di mata (`bg-slate-950` / `bg-dark-900`).
- **Aksen Fungsional**: Warna aksen amber/emas sinematik atau merah brand yang konsisten dan terarah.
- **Micro-Interactions & State Lengkap**: Setiap elemen interaktif (tombol, card, server switcher, episode tab) WAJIB memiliki state yang jelas:
  - `hover:`, `active:`, `focus-visible:ring-2`, dan `disabled:opacity-50`.
  - Transisi halus (`transition duration-200 ease-out`).

### C. Modularitas Template Golang
- Pisahkan komponen UI yang berulang ke dalam sub-template Golang (`{{template "navbar" .}}`, `{{template "movie_card" .}}`).
- Hindari inline styling berantakan; manfaatkan class Tailwind utilitarian yang terstruktur.

