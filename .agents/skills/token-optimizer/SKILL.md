---
name: token-optimizer
description: Penghematan token ekstrem, representasi data format YAML/JSON compact, dan eliminasi zero code boilerplate.
---

# Token Optimizer Skill

## Prinsip Penghematan Token
1. **Representasi Ringkas (Compact Structures)**:
   - Gunakan format YAML atau tabel Markdown padat alih-alih JSON panjang berulang untuk mentransfer state/konteks.
2. **Eliminasi Boilerplate**:
   - Berikan diff atau patch presisi (*surgical edits*) alih-alih menulis ulang seluruh isi file ratusan baris.
3. **Context Trimming**:
   - Buang log verbose dan dependensi yang tidak relevan dengan inti masalah.
   - Gunakan referensi file:// dan simbol fungsi langsung.

