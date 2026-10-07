# 📖 Dokumentasi Lengkap Integrasi API WhatsApp Gateway

Dokumentasi ini menjelaskan cara mengintegrasikan aplikasi backend Anda (Golang, PHP/Laravel, Node.js, Python, dll) dengan **WhatsApp Gateway**.

---

## 🔑 1. Autentikasi API

Setiap request ke endpoint yang memodifikasi data atau mengirim pesan **wajib** menyertakan header API Key:

```http
x-api-key: mysecretkey123
```
*(API Key default adalah `mysecretkey123`. Anda dapat mengubahnya di file `.env`)*

---

## 📱 2. Format Nomor Penerima (`receiver`)

Sistem secara otomatis menyesuaikan format nomor WhatsApp di Indonesia:
- Format lokal: `081234567890` (otomatis diubah ke `6281234567890`)
- Format internasional: `6281234567890`
- Format JID Pengguna: `6281234567890@s.whatsapp.net`
- Format JID Grup: `120363024849204820@g.us`

---

## 📡 3. Daftar Endpoint REST API

Base URL: `http://localhost:8080`

### A. Cek Status Sesi & Koneksi
Mengecek apakah nomor WhatsApp sedang terhubung dan siap digunakan.

- **Method**: `GET`
- **Endpoint**: `/api/session/status`
- **Headers**: *(Publik, tidak butuh API Key)*
- **Contoh Response**:
```json
{
  "success": true,
  "data": {
    "connected": true,
    "logged_in": true,
    "phone": "6283896044531",
    "push_name": "membasuh",
    "jid": "6283896044531:39@s.whatsapp.net",
    "has_qr": false,
    "webhook_url": "http://localhost:8080/webhook/test"
  }
}
```

---

### B. Kirim Pesan Teks
Mengirim pesan teks biasa ke nomor tujuan atau grup WhatsApp.

- **Method**: `POST`
- **Endpoint**: `/api/send/text`
- **Headers**:
  - `Content-Type: application/json`
  - `x-api-key: mysecretkey123`
- **Request Body**:
```json
{
  "receiver": "081234567890",
  "message": "Halo! Pesan ini dikirim secara otomatis melalui WhatsApp Gateway."
}
```
- **Contoh Response Sukses (HTTP 200)**:
```json
{
  "success": true,
  "message": "Pesan berhasil terkirim",
  "data": {
    "id": "3EB0A1B2C3D4E5F6",
    "receiver": "081234567890",
    "timestamp": 1728289900
  }
}
```

---

### C. Kirim Pesan Gambar (Image + Caption)
Mengirim file gambar melalui URL publik beserta teks keterangan (caption).

- **Method**: `POST`
- **Endpoint**: `/api/send/image`
- **Headers**:
  - `Content-Type: application/json`
  - `x-api-key: mysecretkey123`
- **Request Body**:
```json
{
  "receiver": "081234567890",
  "image_url": "https://images.unsplash.com/photo-1579202673506-ca3ce28943ef",
  "caption": "Silahkan cek katalog promo terbaru kami!"
}
```
- **Contoh Response Sukses (HTTP 200)**:
```json
{
  "success": true,
  "message": "Gambar berhasil terkirim",
  "data": {
    "id": "3EB0F9E8D7C6B5A4",
    "receiver": "081234567890",
    "timestamp": 1728289910
  }
}
```

---

### D. Kirim Dokumen / File (PDF, Word, Excel, dll)
Mengirim file dokumen melalui URL publik.

- **Method**: `POST`
- **Endpoint**: `/api/send/document`
- **Headers**:
  - `Content-Type: application/json`
  - `x-api-key: mysecretkey123`
- **Request Body**:
```json
{
  "receiver": "081234567890",
  "document_url": "https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf",
  "filename": "Invoice-INV-2026-001.pdf",
  "caption": "Terlampir struk pembayaran Anda."
}
```
- **Contoh Response Sukses (HTTP 200)**:
```json
{
  "success": true,
  "message": "Dokumen berhasil terkirim",
  "data": {
    "id": "3EB0112233445566",
    "receiver": "081234567890",
    "timestamp": 1728289920
  }
}
```

---

### E. Logout & Reset Sesi
Memutuskan koneksi WhatsApp dari gateway dan menyiapkan QR Code baru.

- **Method**: `POST`
- **Endpoint**: `/api/session/logout`
- **Headers**:
  - `x-api-key: mysecretkey123`
- **Contoh Response**:
```json
{
  "success": true,
  "message": "Berhasil logout. Sesi telah direset dan QR Code baru sedang disiapkan."
}
```

---

## 📥 4. Realtime Incoming Webhook (Pesan Masuk)

Ketika ada pengguna yang mengirim pesan ke nomor WhatsApp Anda, Gateway akan meneruskan request **HTTP POST** secara realtime ke `WEBHOOK_URL` backend Anda.

### Format JSON Webhook:
```json
{
  "event": "message.upsert",
  "timestamp": 1728289500,
  "data": {
    "id": "3EB0ABC12345",
    "from": "628123456789@s.whatsapp.net",
    "sender_name": "Budi Santoso",
    "is_from_me": false,
    "is_group": false,
    "group_jid": "",
    "message_type": "conversation",
    "text": "Halo admin, apakah produk ini masih tersedia?"
  }
}
```

---

## 💻 5. Contoh Kode Integrasi Backend

### 1. Golang

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type SendMessagePayload struct {
	Receiver string `json:"receiver"`
	Message  string `json:"message"`
}

func SendWhatsAppNotification(receiver, text string) error {
	payload := SendMessagePayload{
		Receiver: receiver,
		Message:  text,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "http://localhost:8080/api/send/text", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", "mysecretkey123")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gagal kirim, status code: %d", resp.StatusCode)
	}

	fmt.Println("Pesan WhatsApp berhasil dikirim!")
	return nil
}
```

---

### 2. PHP / Laravel

```php
<?php

namespace App\Services;

use Illuminate\Support\Facades\Http;

class WhatsAppService
{
    protected string $baseUrl = 'http://localhost:8080';
    protected string $apiKey = 'mysecretkey123';

    public function sendText(string $receiver, string $message)
    {
        $response = Http::withHeaders([
            'x-api-key' => $this->apiKey,
        ])->post("{$this->baseUrl}/api/send/text", [
            'receiver' => $receiver,
            'message'  => $message,
        ]);

        return $response->json();
    }

    public function sendImage(string $receiver, string $imageUrl, string $caption = '')
    {
        $response = Http::withHeaders([
            'x-api-key' => $this->apiKey,
        ])->post("{$this->baseUrl}/api/send/image", [
            'receiver'  => $receiver,
            'image_url' => $imageUrl,
            'caption'   => $caption,
        ]);

        return $response->json();
    }
}
```

---

### 3. Node.js (Axios / Fetch)

```javascript
const axios = require('axios');

async function sendWhatsApp(receiver, message) {
  try {
    const response = await axios.post('http://localhost:8080/api/send/text', {
      receiver: receiver,
      message: message,
    }, {
      headers: {
        'Content-Type': 'application/json',
        'x-api-key': 'mysecretkey123',
      },
    });

    console.log('Respon:', response.data);
    return response.data;
  } catch (error) {
    console.error('Gagal kirim pesan:', error.response?.data || error.message);
  }
}
```

---

### 4. Python (Requests)

```python
import requests

def send_whatsapp(receiver: str, message: str):
    url = "http://localhost:8080/api/send/text"
    headers = {
        "Content-Type": "application/json",
        "x-api-key": "mysecretkey123"
    }
    payload = {
        "receiver": receiver,
        "message": message
    }
    response = requests.post(url, json=payload, headers=headers)
    return response.json()
```
