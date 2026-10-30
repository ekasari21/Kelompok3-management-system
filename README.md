# Program Manajemen Sistem Aset dan Komplain (Kelompok 3)

Aplikasi web manajemen aset, keluhan, dan analisis clustering teks berbasis:
- **Backend:** Golang (Gin Framework & GORM)
- **Machine Learning & Analisis Teks:** Python 3 (Scikit-Learn TF-IDF + K-Means Clustering)
- **Frontend:** Vanilla HTML5, CSS3, JavaScript (AJAX)
- **Database:** PostgreSQL (Neon / Supabase TLS)

---

## 🐳 Menjalankan dengan Docker (2 Lingkungan: Go & Python)

Aplikasi ini menggunakan konfigurasi **Multi-stage Docker** yang menggabungkan:
1. Lingkungan **Go 1.24+** untuk kompilasi binary server backend berkinerja tinggi.
2. Lingkungan **Python 3.12** yang dilengkapi library ML (`scikit-learn`, `numpy`) untuk mengeksekusi skrip analisis teks secara otomatis saat endpoint `/api/private/text-analysis` dipanggil.

### Langkah-langkah:

1. **Pastikan file `.env` sudah ada:**
   Buat file `.env` (bisa salin dari `.env.example`):
   ```bash
   cp .env.example .env
   ```
   Pastikan variabel `JWT_SECRET` dan `DATABASE_URL` sudah terisi dengan benar.

2. **Jalankan dengan Docker Compose:**
   ```bash
   docker compose up --build
   ```
   *Atau jika menggunakan docker-compose versi lama:*
   ```bash
   docker-compose up --build
   ```

3. **Akses Aplikasi:**
   - Web App / Landing Page: [http://localhost:8080](http://localhost:8080)
   - Login: [http://localhost:8080/login](http://localhost:8080/login)
   - Register: [http://localhost:8080/register](http://localhost:8080/register)
   - Admin Text Analysis: [http://localhost:8080/client/admin/text-analysis.html](http://localhost:8080/client/admin/text-analysis.html)

4. **Menghentikan Container:**
   ```bash
   docker compose down
   ```

---

## 💻 Menjalankan Secara Lokal (Tanpa Docker)

Jika ingin menjalankan secara langsung di sistem host:

### 1. Prasyarat:
- **Go:** 1.24+ terinstal di sistem
- **Python:** 3.10+ terinstal di sistem

### 2. Instalasi Dependensi Python:
```bash
pip install -r requirements.txt
```

### 3. Konfigurasi `.env`:
Isi file `.env`:
```ini
JWT_SECRET=your_jwt_secret_min_32_bytes
PORT=8080
DATABASE_URL=postgresql://...
# Kosongkan atau arahkan ke python di mesin lokal:
TEXT_ANALYSIS_PYTHON=python
```

### 4. Jalankan Server Go:
```bash
go run .
```
Server akan otomatis mendeteksi executable Python di sistem (PATH, venv, atau instalasi standar) dan menjalankan endpoint text analysis tanpa kendala.
