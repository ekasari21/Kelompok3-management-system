# ==========================================
# STAGE 1: BUILD APLIKASI GO
# ==========================================
FROM golang:1.24-bookworm AS builder

WORKDIR /app

# Salin dependency file Go
COPY go.mod go.sum ./

# Download dependensi Go (memanfaatkan cache layer Docker)
RUN go mod download

# Salin seluruh source code proyek
COPY . .

# Kompilasi binary Go secara statis untuk arsitektur Linux
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server .


# ==========================================
# STAGE 2: RUNTIME 2 LINGKUNGAN (GO + PYTHON)
# ==========================================
FROM python:3.12-slim-bookworm

WORKDIR /app

# Instal ca-certificates agar koneksi TLS/SSL ke database PostgreSQL (Supabase/Neon) aman dan valid
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Salin dan instal dependensi Python (Machine Learning: scikit-learn & numpy)
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Verifikasi library Python yang digunakan modul Text Analysis
RUN python -c "import numpy, sklearn, pandas, scipy; print('Python ML dependencies OK')"

# Salin binary aplikasi Go dari Stage 1
COPY --from=builder /app/server /app/server

# Salin file statis frontend, internal package/scripts, dan template folder
COPY frontend ./frontend
COPY uploads ./uploads
COPY internal ./internal

# Pastikan folder penyimpanan uploads dan dashboard tersedia
RUN mkdir -p uploads/complaints internal/dashboard

# Konfigurasi environment runtime
ENV PORT=8080 \
    GIN_MODE=release \
    TEXT_ANALYSIS_PYTHON=python3 \
    TEXT_ANALYSIS_SCRIPT=internal/textanalysis/analyze_text.py \
    PYTHONUNBUFFERED=1

# Expose port aplikasi
EXPOSE 8080

# Volume untuk upload file gambar agar persisten
VOLUME ["/app/uploads"]

# Jalankan backend Go yang mengeksekusi endpoint REST API sekaligus modul Python
CMD ["/app/server"]
