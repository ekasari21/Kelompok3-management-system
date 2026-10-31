# 🏢 Sistem Manajemen Aset, Komplain & Analisis Teks (Kelompok 3)

Sistem informasi berbasis web yang dirancang untuk mengelola inventaris aset, peminjaman barang, pelaporan kerusakan, pengaduan keluhan (*complaints*), serta analisis klasterisasi teks otomatis berbasis Machine Learning.

Aplikasi ini mengintegrasikan **dua lingkungan komputasi**:
- **Backend:** Golang (Gin & GORM) untuk pemrosesan API dan transaksi data berkecepatan tinggi.
- **Machine Learning:** Python 3 (Scikit-Learn) untuk analisis pengelompokan teks keluhan dan kerusakan (*TF-IDF & K-Means Clustering*).

---

## 📑 Daftar Isi
- [Teknologi yang Digunakan](#-teknologi-yang-digunakan)
- [Fitur Utama](#-fitur-utama)
- [Struktur Proyek](#-struktur-proyek)
- [Panduan Instalasi & Menjalankan](#-panduan-instalasi--menjalankan)
  - [Metode 1: Menggunakan Docker (Go + Python Otomatis)](#metode-1-menggunakan-docker-rekomendasi)
  - [Metode 2: Menjalankan Secara Lokal](#metode-2-menjalankan-secara-lokal)
- [Konfigurasi Environment Variable (`.env`)](#-konfigurasi-environment-variable-env)
- [Dokumentasi Endpoint API](#-dokumentasi-endpoint-api)
- [Hak Akses Pengguna (Role)](#-hak-akses-pengguna-role)
- [Troubleshooting](#-troubleshooting)

---

## 🛠 Teknologi yang Digunakan

| Kategori | Teknologi | Keterangan |
| :--- | :--- | :--- |
| **Backend API** | [Go (Golang)](https://go.dev/) 1.24+ | RESTful API menggunakan **Gin Gonic** & **GORM** |
| **Keamanan** | JWT & Bcrypt | Autentikasi berbasis JSON Web Token (HttpOnly Cookie) & hashing password |
| **Text Analysis & Clustering** | Native Go (Golang) | Algoritma *TF-IDF Vectorizer* & *K-Means Clustering* diimplementasikan murni dalam bahasa Go |
| **Database** | PostgreSQL | Cloud Database (Neon / Supabase) dengan koneksi aman TLS/SSL |
| **Frontend** | Vanilla JS, HTML5, CSS3, Bootstrap 5 | Antarmuka responsif dengan AJAX Fetch, SweetAlert2, dan Chart.js |
| **Containerization** | Docker & Docker Compose | Multi-stage build yang menggabungkan binary Go dan runtime Python |

---

## ✨ Fitur Utama

1. **🔐 Autentikasi & Multi-Role:**
   - Registrasi, Login, dan Logout menggunakan cookie aman.
   - Pembagian hak akses: **Admin**, **Teknisi**, dan **User**.
2. **📦 Manajemen Master Data:**
   - **Aset:** Pendataan barang, kode barcode/seri, spesifikasi, jumlah stok, kategori, dan lokasi.
   - **Kategori:** Pengelompokan jenis aset (elektronik, furnitur, dsb.).
   - **Lokasi:** Ruangan atau gedung penempatan aset.
3. **🔄 Transaksi Aset:**
   - **Peminjaman Aset (`asset-lending`):** Pengajuan peminjaman barang, batas waktu pinjam, dan konfirmasi pengembalian.
   - **Aset Rusak (`asset-broken`):** Pencatatan unit aset yang mengalami kerusakan serta status kelayakan perbaikan.
4. **📢 Sistem Keluhan / Komplain (`complaints`):**
   - User dapat mengirimkan laporan komplain lengkap dengan bukti foto fisik.
   - Pelacakan tingkat urgensi (*low, medium, high, critical*) dan status tindak lanjut oleh teknisi.
5. **📊 Dashboard Interaktif:**
   - Ringkasan statistik aset tersedia, dipinjam, rusak, dan total komplain.
   - Log histori aktivitas pengguna secara real-time (`activities.json`).
6. **🤖 Machine Learning Text Analysis:**
   - Menganalisis judul dan deskripsi keluhan atau kerusakan secara otomatis.
   - Menghasilkan klaster masalah serupa, ringkasan kata kunci dominan (*top keywords*), dan estimasi total unit terdampak tanpa pelabelan manual.
7. **👥 Manajemen Pengguna:**
   - Pengelolaan akun pengguna dan penggantian kata sandi oleh Administrator.
8. **💬 Asisten Virtual Inventaris (Chatbot):**
   - Asisten AI cerdas untuk konsultasi data inventaris (stok aset, ketersediaan, peminjaman barang, aset rusak, kategori, dan lokasi ruangan).
   - **Aturan Ketat:** Chatbot dibatasi secara ketat (*domain guardrail*) di mana bot **hanya menjawab seputar data inventaris**. Pertanyaan di luar data inventaris (politik, cuaca, hiburan, resep masakan, coding umum, lelucon, dll.) akan ditolak secara sopan.
   - Terintegrasi langsung dengan database PostgreSQL secara real-time dan dilengkapi rekomendasi pertanyaan cepat.

---

## 📂 Struktur Proyek

```text
Kelompok3-management-system/
├── frontend/                   # File antarmuka statis
│   ├── index.html              # Landing page
│   ├── login.html              # Halaman login
│   ├── register.html           # Halaman pendaftaran
│   └── client/                 # Dashboard & modul klien
│       ├── admin/              # Halaman dan AJAX khusus Admin
│       ├── teknisi/            # Halaman dan AJAX khusus Teknisi
│       ├── user/               # Halaman dan AJAX khusus User
│       ├── css/                # Stylesheet
│       └── js/                 # Skrip helper & auth layout
├── internal/                   # Kode modul backend (Go)
│   ├── asset/                  # Logika CRUD aset
│   ├── assetbroken/            # Logika pelaporan aset rusak
│   ├── assetlending/           # Logika peminjaman & pengembalian
│   ├── auth/                   # Autentikasi, JWT, & registrasi
│   ├── category/               # Master kategori
│   ├── chatbot/                # Modul Asisten Virtual Inventaris (NLP & Repository)
│   ├── complaint/              # Pelaporan komplain & upload file
│   ├── dashboard/              # Ringkasan analitik & histori aktivitas
│   ├── database/               # Koneksi PostgreSQL (GORM)
│   ├── location/               # Master lokasi
│   ├── textanalysis/           # Modul Native Go Text Analysis (TF-IDF & K-Means)
│   └── user/                   # Manajemen pengguna
├── uploads/                    # Folder penyimpanan file upload (foto komplain)
├── Dockerfile                  # Konfigurasi container Go + Python
├── docker-compose.yml          # Orkestrasi container Docker
├── .dockerignore               # Filter file context build Docker
├── requirements.txt            # Dependensi library Python
├── go.mod & go.sum             # Dependensi modul Go
├── main.go                     # Entry point server web Gin
├── .env.example                # Template variabel lingkungan
└── README.md                   # Dokumentasi proyek
```

---

## 🚀 Panduan Instalasi & Menjalankan

### Metode 1: Menggunakan Docker (Rekomendasi)

Pendekatan ini secara otomatis menyiapkan **Go runtime** dan **Python ML runtime** di dalam satu kontainer tanpa memerlukan instalasi manual dependensi di komputer Anda.

#### 1. Persiapkan File Konfigurasi:
Salin file `.env.example` menjadi `.env`:
```bash
cp .env.example .env
```
Pastikan `DATABASE_URL` dan `JWT_SECRET` pada file `.env` sudah terisi dengan benar.

#### 2. Jalankan Container:
```bash
docker compose up --build
```
> **Catatan:** Tambahkan parameter `-d` jika ingin menjalankan di latar belakang (*detached mode*): `docker compose up --build -d`.

#### 3. Akses Aplikasi:
- **Landing Page:** [http://localhost:8080](http://localhost:8080)
- **Login:** [http://localhost:8080/login](http://localhost:8080/login)
- **Register:** [http://localhost:8080/register](http://localhost:8080/register)

#### 4. Menghentikan Container:
```bash
docker compose down
```

---

### Metode 2: Menjalankan Secara Lokal

Gunakan metode ini jika ingin mengembangkan langsung di sistem host.

#### 1. Prasyarat:
- **Go:** Versi 1.24 atau yang lebih baru ([Download Go](https://go.dev/dl/))
- **Python:** Versi 3.10 atau yang lebih baru ([Download Python](https://www.python.org/downloads/))
- **Database:** Akun PostgreSQL aktif (misal dari Neon.tech atau Supabase)

#### 2. Instalasi Dependensi Python:
Buka terminal dan instal library Machine Learning:
```bash
pip install -r requirements.txt
```

#### 3. Unduh Modul Go:
```bash
go mod download
```

#### 4. Konfigurasi `.env`:
Pastikan file `.env` sudah ada dan arahkan variabel Python ke executable di komputer Anda (atau kosongkan agar server mendeteksi otomatis dari PATH):
```ini
JWT_SECRET=rahasia_jwt_minimal_32_karakter_acak
PORT=8080
DATABASE_URL='postgresql://user:password@host:5432/dbname?sslmode=require'

# Di Windows bisa diisi nama command atau path absolut python.exe:
TEXT_ANALYSIS_PYTHON=python
```

#### 5. Jalankan Server:
```bash
go run .
```

---

## ⚙ Konfigurasi Environment Variable (`.env`)

| Variabel | Wajib | Deskripsi | Contoh |
| :--- | :---: | :--- | :--- |
| `PORT` | Tidak | Port HTTP server Go (default: `8080`) | `8080` |
| `JWT_SECRET` | **Ya** | Kunci enkripsi token JWT (minimal 32 byte) | `8cNp4YOF4Sq8b1QJdtkXfxaQC7iK1lKQbBB3VKZEo1A=` |
| `DATABASE_URL` | **Ya** | Connection string PostgreSQL (wajib SSL/TLS) | `postgresql://user:pass@host:5432/db?sslmode=require` |
| `TEXT_ANALYSIS_PYTHON` | Opsional | Path interpreter Python untuk Text Analysis | `python3` (Docker/Linux) atau path ke `python.exe` |
| `TEXT_ANALYSIS_SCRIPT` | Opsional | Lokasi skrip Python Text Analysis | `internal/textanalysis/analyze_text.py` |
| `ALLOWED_ORIGIN` | Opsional | Origin CORS tambahan | `http://localhost:8081` |

---

## 📡 Dokumentasi Endpoint API

Semua endpoint dengan prefix `/api/private/*` membutuhkan autentikasi melalui Cookie `access_token` JWT.

### 🔑 Autentikasi (`/api/auth`)
| Method | Endpoint | Deskripsi | Akses |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/auth/register` | Mendaftarkan akun baru | Publik |
| `POST` | `/api/auth/login` | Autentikasi pengguna & pembuatan cookie sesi | Publik |
| `POST` | `/api/auth/logout` | Menghapus sesi dan cookie login | Publik |
| `GET` | `/api/me` | Mengambil data pengguna yang sedang login | Terautentikasi |

### 📂 Kategori (`/api/private/categories`)
| Method | Endpoint | Deskripsi |
| :--- | :--- | :--- |
| `GET` | `/api/private/categories` | Ambil semua daftar kategori |
| `POST` | `/api/private/categories` | Tambah kategori baru |
| `GET` | `/api/private/categories/:id` | Detail kategori |
| `PUT` | `/api/private/categories/:id` | Perbarui nama kategori |
| `DELETE` | `/api/private/categories/:id` | Hapus kategori |

### 📍 Lokasi (`/api/private/locations`)
| Method | Endpoint | Deskripsi |
| :--- | :--- | :--- |
| `GET` | `/api/private/locations` | Ambil semua daftar lokasi |
| `POST` | `/api/private/locations` | Tambah lokasi penempatan baru |
| `GET` | `/api/private/locations/:id` | Detail lokasi |
| `PUT` | `/api/private/locations/:id` | Perbarui data lokasi |
| `DELETE` | `/api/private/locations/:id` | Hapus lokasi |

### 💬 Asisten Virtual Inventaris / Chatbot (/api/private/chatbot)
| Method | Endpoint | Deskripsi |
| :--- | :--- | :--- |
| POST | /api/private/chatbot/ask | Mengirim pertanyaan dan menerima jawaban seputar data inventaris |
| GET | /api/private/chatbot/suggestions | Mendapatkan daftar saran pertanyaan cepat seputar inventaris |

### 📦 Aset (`/api/private/assets`)
| Method | Endpoint | Deskripsi |
| :--- | :--- | :--- |
| `GET` | `/api/private/assets` | Ambil semua daftar aset |
| `POST` | `/api/private/assets` | Tambah inventaris aset baru |
| `GET` | `/api/private/assets/:id` | Detail aset |
| `PUT` | `/api/private/assets/:id` | Perbarui informasi aset |
| `DELETE` | `/api/private/assets/:id` | Hapus aset (*soft-delete*) |

### ⚠️ Aset Rusak (`/api/private/asset-broken`)
| Method | Endpoint | Deskripsi |
| :--- | :--- | :--- |
| `GET` | `/api/private/asset-broken` | Ambil daftar laporan aset rusak |
| `POST` | `/api/private/asset-broken` | Laporkan unit aset rusak |
| `GET` | `/api/private/asset-broken/:id` | Detail laporan kerusakan |
| `PUT` | `/api/private/asset-broken/:id` | Perbarui status dan kelayakan perbaikan |
| `DELETE` | `/api/private/asset-broken/:id` | Hapus data kerusakan aset |

### 🤝 Peminjaman Aset (`/api/private/asset-lending`)
| Method | Endpoint | Deskripsi |
| :--- | :--- | :--- |
| `GET` | `/api/private/asset-lending` | Ambil riwayat peminjaman aset |
| `POST` | `/api/private/asset-lending` | Ajukan peminjaman aset |
| `GET` | `/api/private/asset-lending/:id` | Detail peminjaman aset |
| `PUT` | `/api/private/asset-lending/:id` | Ubah status / data peminjaman |
| `PUT` | `/api/private/asset-lending/:id/return` | Proses pengembalian aset yang dipinjam |
| `DELETE` | `/api/private/asset-lending/:id` | Hapus riwayat peminjaman |

### 📢 Keluhan / Komplain (`/api/private/complaints`)
| Method | Endpoint | Deskripsi |
| :--- | :--- | :--- |
| `GET` | `/api/private/complaints` | Ambil daftar seluruh keluhan |
| `POST` | `/api/private/complaints` | Buat laporan keluhan baru (upload multipart) |
| `GET` | `/api/private/complaints/:id` | Detail keluhan & foto bukti |
| `PUT` | `/api/private/complaints/:id` | Update status tindak lanjut keluhan |
| `DELETE` | `/api/private/complaints/:id` | Hapus keluhan |

### 📊 Dashboard & Analitik (`/api/private/dashboard`)
| Method | Endpoint | Deskripsi |
| :--- | :--- | :--- |
| `GET` | `/api/private/dashboard/summary` | Statistik ringkasan aset, komplain, peminjaman |
| `GET` | `/api/private/dashboard/lastAsset` | Daftar 5 aset yang baru ditambahkan |
| `GET` | `/api/private/dashboard/activities` | Histori log 5 aktivitas terbaru sistem |

### 🧠 Analisis Teks Machine Learning (`/api/private/text-analysis`)
| Method | Endpoint | Parameter Query | Deskripsi |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/private/text-analysis` | `source=all\|complaints\|damage`<br>`clusters=2..10` | Menjalankan klasterisasi otomatis K-Means dan ekstraksi TF-IDF pada teks komplain/kerusakan melalui subprocess Python. |

### 👥 Pengguna (`/api/private/users`)
| Method | Endpoint | Deskripsi |
| :--- | :--- | :--- |
| `GET` | `/api/private/users` | Ambil daftar seluruh pengguna (Admin) |
| `POST` | `/api/private/users` | Tambah pengguna baru langsung dari admin |
| `PUT` | `/api/private/users/:id/password` | Perbarui kata sandi pengguna |

---

## 👥 Hak Akses Pengguna (Role)

1. **Admin (`admin`):**
   - Akses penuh ke seluruh menu dan data master.
   - Mengelola aset, kategori, lokasi, peminjaman, komplain, dan manajemen akun pengguna.
   - Mengakses modul Text Analysis & Machine Learning.
2. **Teknisi (`teknisi`):**
   - Melihat dan memperbarui status komplain dari pengguna.
   - Mencatat dan memperbarui status perbaikan pada unit aset rusak.
3. **User (`user`):**
   - Melihat daftar aset yang tersedia.
   - Mengajukan peminjaman aset.
   - Membuat laporan keluhan / komplain kerusakan aset.

---

## ❓ Troubleshooting

### 1. Error: `analisis Python gagal: exec: "...": executable file not found in $PATH`
- **Penyebab:** Sistem mencoba mencari interpreter Python pada path yang tidak valid.
- **Solusi:**
  - Jika menggunakan **Docker**, gunakan `docker compose up --build` karena `TEXT_ANALYSIS_PYTHON: "python3"` sudah otomatis terkonfigurasi.
  - Jika berjalan secara **Lokal Windows**, pastikan Python terpasang di PATH atau atur di file `.env`:
    ```ini
    TEXT_ANALYSIS_PYTHON=python
    ```
    atau gunakan path lengkap:
    ```ini
    TEXT_ANALYSIS_PYTHON=C:\Users\NamaUser\AppData\Local\Programs\Python\Python313\python.exe
    ```

### 2. Error TLS / SSL Database (`x509: certificate signed by unknown authority`)
- **Penyebab:** CA certificates belum terpasang di container atau koneksi internet terblokir.
- **Solusi:** `Dockerfile` terbaru sudah menyertakan instalasi paket `ca-certificates`. Jalankan build ulang kontainer dengan `docker compose build --no-cache`.

### 3. Port 8080 Bentrok dengan Aplikasi Lain
- Ganti port pada file `.env` atau `docker-compose.yml`:
  ```ini
  PORT=8082
  ```
  Lalu akses aplikasi melalui `http://localhost:8082`.
