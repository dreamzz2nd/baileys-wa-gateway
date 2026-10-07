# 🚀 WhatsApp Gateway (Go + Whatsmeow)

WhatsApp Gateway ultra-ringan berbasis **Golang** dan **Whatsmeow** dengan database SQLite Pure Go (tanpa butuh CGO / GCC). Memakan memori RAM sangat kecil (**~20 MB RAM**).

---

## 🎨 Tampilan Dashboard Web UI (Tema Putih Bersih)

Buka di browser:
👉 **`http://localhost:8080`**

Fitur Web UI:
- **Scan QR Code & Monitor Sesi**: Realtime QR Code dan status akun WhatsApp yang terhubung.
- **Form Test Kirim Pesan**: Coba kirim pesan teks secara langsung dari browser.
- **Live Webhook Viewer**: Log pesan masuk ditampilkan secara realtime.
- **Dokumentasi & Code Snippet**: Salin contoh kode integrasi (cURL, Go, Laravel, Node.js, Python).

---

## 🏃‍♂️ Cara Menjalankan

```bash
cd go-gateway
go run main.go
```

Atau jalankan binary `.exe` yang sudah di-compile:

```bash
./wa-gateway.exe
```

---

## 📖 Dokumentasi Lengkap API & Integrasi

Panduan lengkap integrasi API, parameter request/response, format webhook, dan contoh kode backend tersedia di:
👉 **[API_DOCUMENTATION.md](file:///c:/Users/user/wa-gateway/Baileys/go-gateway/API_DOCUMENTATION.md)**

---

## ⚡ Ringkasan Endpoint

| Method | Endpoint | Deskripsi | Auth Header |
|---|---|---|---|
| `GET` | `/api/session/status` | Cek status & info akun terhubung | Publik |
| `GET` | `/api/session/qr` | Mengambil data QR code | Publik |
| `POST` | `/api/send/text` | Kirim pesan teks | `x-api-key: mysecretkey123` |
| `POST` | `/api/send/image` | Kirim gambar via URL + caption | `x-api-key: mysecretkey123` |
| `POST` | `/api/send/document` | Kirim file / PDF via URL | `x-api-key: mysecretkey123` |
| `POST` | `/api/session/logout` | Logout sesi WhatsApp | `x-api-key: mysecretkey123` |

---

## 📥 Webhook Pesan Masuk

Atur `WEBHOOK_URL` di file `.env` (misal `http://localhost:3000/webhook`). Setiap ada pesan masuk ke WhatsApp, gateway akan otomatis melakukan `POST` payload JSON ke backend Anda.
