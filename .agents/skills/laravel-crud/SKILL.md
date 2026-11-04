---
name: laravel-crud
description: Pengembangan backend, Eloquent ORM, FormRequest validation, arsitektur Service-Repository, dan Admin Panel.
---

# Laravel CRUD Mastery Skill

## Cakupan & Panduan
1. **Model & Eloquent**:
   - Gunakan `$fillable` / `$guarded` dengan aman.
   - Definisi relasi (`hasMany`, `belongsTo`, `belongsToMany`) dengan type hinting PHP 8.2+.
   - Manfaatkan Eloquent Scopes dan Accessors/Mutators modern (`Attribute::make()`).
2. **FormRequest Validation**:
   - Selalu pisahkan aturan validasi ke FormRequest (`php artisan make:request`).
   - Terapkan `authorize()` dan `rules()` yang presisi dengan pesan localized.
3. **Admin Panel & Resource Controller**:
   - Controller ramping: delegasikan logika bisnis ke Action/Service classes.
   - Resourceful controller (`index`, `create`, `store`, `show`, `edit`, `update`, `destroy`).
