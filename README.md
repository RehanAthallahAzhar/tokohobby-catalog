# 🛒 Shopeezy

## Business Understanding
Dalam sistem e-commerce modern, skala besar dan kompleksitas bisnis menuntut arsitektur yang fleksibel, terukur, dan tahan terhadap kegagalan. Untuk itu, arsitektur microservice dipilih guna memisahkan domain bisnis seperti pengguna, produk, keranjang, dan pesanan menjadi service independen.

## Permasalahan Bisnis
Sistem monolitik sulit di-maintain karena:
- Sulit diskalakan per fitur.
- Satu kegagalan dapat melumpuhkan seluruh sistem.
- Proses checkout rentan terhadap bottleneck, race condition, dan coupling antar modul.

## Tujuan
- Membangun sistem e-commerce berbasis microservice yang modular dan scalable.
- Menggunakan gRPC untuk komunikasi sinkron dan RabbitMQ untuk komunikasi asinkron (event-driven).
- Memastikan alur checkout efisien, aman dari race condition, dan resilient terhadap failure antar service.


## Struktur Proyek
1.  **shopeezy-protos**

    ```
    proto/
    └── auth/
        └── auth.proto  # Definisi service Auth gRPC
    ```

2. **shopeezy-account**

    ```
    account/
    ├── .env # Konfigurasi environment (DB, Redis, dll)
    ├── main.go # Entry point utama
    ├── databases/
    │   └── db.go # Koneksi dan inisialisasi database
    ├── models/
    │   └── user.go # Model untuk entitas User
    ├── pkg/
    │   └── redisclient/
    │       └── redis_client.go # Inisialisasi koneksi Redis
    ├── repositories/
    │   ├── user_repository.go # Akses data user
    │   └── jwt_blacklist_repository.go # Akses blacklist token JWT
    ├── services/
    │   ├── token_service.go # Interface untuk JWT token
    │   └── jwt_token_service.go # Implementasi token JWT (generate/validate)
    ├── internal/
    │   └── grpc/
    │       └── auth_server.go # Implementasi gRPC Auth service
    ├── handlers/
    │   └── api.go # Handler untuk REST API
    ├── middlewares/
    │   └── auth_middleware.go # Middleware otorisasi REST API
    └── routes/
        └── routes.go # Setup routing REST API
    ```

3. **shopeezy-inventory-cart**

    ```
    inventory-cart/
    ├── .env                         # Konfigurasi environment
    ├── main.go                      # Entry point utama
    ├── databases/
    │   └── db.go                    # Koneksi database
    ├── pkg/
    │   ├── redisclient/
    │   │   └── redis_client.go      # Koneksi Redis
    │   └── authclient/
    │       └── auth_client.go       # gRPC client ke service Auth
    ├── models/
    │   └── cart.go                  # Model: Product, CartItem, Order, OrderItem
    ├── repositories/
    │   ├── product_repository.go    # Akses data produk
    │   └── cart_repository.go       # Akses data keranjang & pesanan
    ├── handlers/
    │   └── api.go                   # Handler untuk REST API
    └── routes/
        └── routes.go                # Setup routing REST API
    ```

## Proposal Fitur Inti 

Proyek ini mendemonstrasikan arsitektur layanan mikro dengan fokus pada tiga layanan utama dan interaksinya. Kami akan mengeksplorasi manajemen pengguna, inventaris, keranjang belanja, proses *checkout*, dan notifikasi, semua diorkestrasi melalui API REST, gRPC, dan *message queuing* menggunakan RabbitMQ.

---

### 1. `account-cashier-app-v2` (Layanan Akun)

Layanan ini bertanggung jawab atas manajemen pengguna dan autentikasi.

#### Fungsi Utama
* **Manajemen Pengguna**: Pendaftaran, *login*, dan pengelolaan profil pengguna.
* **Autentikasi JWT**: Penerbitan dan validasi token JSON Web Token.

#### Teknologi
* **Go**: Bahasa pemrograman utama.
* **Echo**: *Web framework* untuk API REST.
* **PostgreSQL**: Basis data relasional untuk penyimpanan data pengguna.
* **JWT**: Untuk autentikasi.
* **Redis**: Digunakan sebagai *blacklist* untuk token JWT yang sudah dicabut.

#### Interaksi
* **REST API**: Menyediakan *endpoint* untuk *frontend* (misalnya, `/register`, `/login`).
* **gRPC API**: Menyediakan antarmuka untuk layanan lain guna memvalidasi token autentikasi.
* **BARU - Konsumsi Event RabbitMQ**: Mengonsumsi *event* dari RabbitMQ untuk pemrosesan asinkron, contohnya mendebit saldo pengguna setelah sebuah pesanan berhasil.

---

### 2. `cashier-app` (Layanan Inventaris & Keranjang)

Layanan ini menangani manajemen produk, fungsionalitas keranjang belanja, dan proses *checkout*.

#### Fungsi Utama
* **Manajemen Produk**: Operasi CRUD (Create, Read, Update, Delete) untuk produk.
* **Manajemen Keranjang Belanja**: Menambah, menghapus, dan memperbarui item di keranjang.
* **Proses Checkout**: Mengelola alur pembelian produk.

#### Teknologi
* **Go**: Bahasa pemrograman utama.
* **Echo**: *Web framework* untuk API REST.
* **PostgreSQL**: Basis data relasional untuk penyimpanan data produk dan keranjang.
* **Redis**: Digunakan untuk *caching* data produk dan keranjang belanja guna meningkatkan kinerja.

#### Interaksi
* **REST API**: Menyediakan *endpoint* untuk *frontend* (misalnya, `/products`, `/cart`, `/checkout`).
* **Konsumsi gRPC API**: Mengonsumsi gRPC API dari `account-cashier-app-v2` untuk memvalidasi token pengguna yang mengakses layanan ini.
* **BARU - Publikasi Event RabbitMQ**: Mempublikasikan *event* ke RabbitMQ setelah sebuah pesanan berhasil diselesaikan (setelah proses *checkout*).

---

### 3. Layanan Notifikasi Sederhana (Notification Service)

Layanan ringan ini bertanggung jawab untuk mensimulasikan pengiriman notifikasi.

#### Fungsi Utama
* **Simulasi Notifikasi**: Mensimulasikan pengiriman notifikasi, seperti email konfirmasi pesanan.

#### Teknologi
* **Go**: Bahasa pemrograman yang digunakan, dirancang agar sangat ringan (mungkin hanya dalam satu file `main.go`).

#### Interaksi
* **BARU - Konsumsi Event RabbitMQ**: Mengonsumsi *event* yang dipublikasikan oleh `cashier-app` (misalnya, *event* "pesanan berhasil") untuk memicu pengiriman notifikasi.

Interaksi:

- Menyediakan REST API untuk frontend (login, register).

- Menyediakan gRPC API untuk validasi token oleh layanan lain.

- Mengonsumsi event RabbitMQ untuk pemrosesan asinkron (misalnya, mendebit saldo setelah pesanan).


Untuk Detailnya bisa klik disini

## User Story
google drive user story








## References

**folder pattern** : <https://github.com/restuwahyu13/go-clean-architecture>
