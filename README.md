# Ayam Batu Backend API (Golang + Gin + GORM + PostgreSQL)

Backend REST API untuk manajemen produk, pencatatan harga beli per batch, pelacakan stok (dibeli, terjual, sisa), perhitungan harga jual minimal otomatis, pencatatan penjualan harian, laporan keuangan (Modal, Bruto, Netto), serta autentikasi Sign Up & Login menggunakan JWT.

## Fitur Utama

1. **Autentikasi (Sign Up & Login)**
   - Password di-hash menggunakan `bcrypt`.
   - Autentikasi API menggunakan JWT Bearer Token.
2. **Manajemen Produk (CRUD)**
   - Tambah, lihat, ubah, dan hapus produk.
   - Setiap produk otomatis menampilkan ringkasan stok (`total_bought_stock`, `total_sold_stock`, `remaining_stock`) serta `min_sell_price`.
3. **Pencatatan Harga Beli & Stok per Batch (CRUD)**
   - Catat `buy_price`, `initial_stock` (stok dibeli), `sold_stock` (stok terjual), dan `remaining_stock` (stok sisa) per batch.
   - Validasi agar `initial_stock` tidak bisa diubah lebih kecil dari `sold_stock` dan batch yang sudah terjual tidak bisa dihapus sembarangan.
4. **Harga Jual Minimal (Simple Average)**
   - Dihitung dari rata-rata harga beli seluruh batch produk tersebut (`AVG(buy_price)`), sehingga penjualan di atas/sama dengan nilai ini menjaga agar tidak rugi.
   - Endpoint pembuatan/edit penjualan menolak harga jual di bawah `min_sell_price` secara default (kecuali `allow_below_min: true` dikirim eksplisit).
5. **Pencatatan Produk Terjual Per Hari (CRUD + FIFO)**
   - Mencatat transaksi penjualan harian (`sold_at` format `YYYY-MM-DD`).
   - Stok dipotong secara **FIFO** (batch tertua lebih dulu) dan dicatat di `sale_batch_allocations`, sehingga update/hapus transaksi penjualan otomatis mengembalikan stok batch dengan akurat.
   - Tersedia endpoint rekap penjualan harian (`GET /api/v1/sales/daily`).
6. **Laporan Keuangan (Modal, Bruto, Netto)**
   - `modal`: total modal (HPP) dari produk yang terjual berdasarkan alokasi harga beli batch.
   - `bruto`: total omzet kotor penjualan (`quantity * sell_price`).
   - `netto`: laba bersih (`bruto - modal`).
   - `total_purchase_modal`: total nilai modal dari seluruh batch pembelian.

---

## Menjalankan Proyek

### 1. Dengan Docker Compose (PostgreSQL + API)

```bash
cp .env.example .env
docker compose up -d --build
```

### 2. Menjalankan PostgreSQL via Docker & API Secara Lokal

```bash
cp .env.example .env
docker compose up -d postgres
go run ./cmd/api
```

Migrasi SQL di [migrations/001_init.sql](migrations/001_init.sql) otomatis dijalankan saat startup aplikasi.

---

## Daftar Endpoint

Base URL: `http://localhost:8080/api/v1`

### Auth
- `POST /auth/signup` — Daftar akun baru (`name`, `email`, `password`)
- `POST /auth/login` — Login (`email`, `password`) -> mengembalikan JWT `token`
- `GET /auth/google/login` — Redirect ke halaman login Google OAuth 2.0 *(buka di browser)*
- `GET /auth/google/callback` — Callback Google OAuth 2.0 -> menerbitkan JWT `token` (atau redirect ke `GOOGLE_FRONTEND_REDIRECT_URL?token=...` jika diisi)
- `GET /auth/me` — Profil user login *(Bearer Token)*

### Products *(Bearer Token)*
- `POST /products` — Tambah produk (`name`, `sku`, `unit`, `description`)
- `GET /products` — List produk + ringkasan stok & `min_sell_price`
- `GET /products/:id` — Detail produk + daftar batch
- `PUT /products/:id` — Update produk
- `DELETE /products/:id` — Hapus produk

### Stock Batches *(Bearer Token)*
- `POST /batches` — Tambah batch pembelian (`product_id`, `batch_code`, `buy_price`, `initial_stock`, `purchased_at`, `notes`)
- `GET /batches?product_id=1` — List batch (opsional filter `product_id`)
- `GET /batches/:id` — Detail batch
- `PUT /batches/:id` — Update batch
- `DELETE /batches/:id` — Hapus batch (hanya jika `sold_stock == 0`)

### Sales *(Bearer Token)*
- `POST /sales` — Catat penjualan (`product_id`, `quantity`, `sell_price`, `sold_at`, `notes`, `allow_below_min`)
- `GET /sales?product_id=1&date=2026-11-26` — List transaksi penjualan (filter: `product_id`, `date`, `start_date`, `end_date`)
- `GET /sales/daily?start_date=2026-11-01&end_date=2026-11-30` — Rekap produk terjual per hari
- `GET /sales/:id` — Detail transaksi & alokasi batch
- `PUT /sales/:id` — Update transaksi penjualan (otomatis hitung ulang stok & keuangan)
- `DELETE /sales/:id` — Hapus transaksi penjualan (otomatis kembalikan stok batch)

### Financial Reports *(Bearer Token)*
- `GET /reports/financial?start_date=2026-11-01&end_date=2026-11-30&product_id=1` — Laporan `modal`, `bruto`, `netto`, `total_purchase_modal`, stok, dan rincian per produk
- `GET /reports/daily?start_date=2026-11-01&end_date=2026-11-30` — Laporan `modal`, `bruto`, `netto` harian

---

## Dokumentasi API & Cara Testing

### 1. Dokumentasi Interaktif (Swagger UI & OpenAPI)
Saat server berjalan, buka browser ke:
- **Swagger UI**: `http://localhost:8080/docs`
- **OpenAPI 3.0 Spec**: `http://localhost:8080/docs/openapi.yaml` atau file [docs/openapi.yaml](docs/openapi.yaml)

Di Swagger UI:
1. Jalankan `POST /api/v1/auth/signup` atau `POST /api/v1/auth/login` untuk mendapatkan `token`.
2. Klik tombol **Authorize** di kanan atas, lalu masukkan `token` JWT Anda (cukup paste token-nya).
3. Semua endpoint di bawah Products, Batches, Sales, dan Reports bisa langsung diuji lewat tombol **Try it out**.

### 2. Testing via Postman
Import file [docs/postman_collection.json](docs/postman_collection.json) ke Postman.
- Request **Sign Up** dan **Login** sudah dilengkapi test script otomatis untuk menyimpan `token` ke variable `{{token}}`.
- Request **Create Product**, **Create Batch**, dan **Create Daily Sale** juga otomatis menyimpan `product_id`, `batch_id`, dan `sale_id`.

### 3. Unit Test (Tanpa Database)
```powershell
go test -v ./...
```

### 4. End-to-End Smoke Test (Dengan PostgreSQL + Server Aktif)
Setelah menjalankan `docker compose up -d postgres` dan `go run ./cmd/api`, jalankan script PowerShell berikut untuk menguji seluruh alur CRUD (Sign Up -> Login -> Tambah Produk -> Tambah 2 Batch -> Cek `min_sell_price` -> Uji tolak harga di bawah minimal -> Catat Penjualan -> Cek Laporan Modal/Bruto/Netto -> Hapus Penjualan & cek stok kembali):

```powershell
.\scripts\test_api.ps1
```
