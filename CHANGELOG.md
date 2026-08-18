# Changelog

Semua perubahan penting KasQ dicatat di sini.

Format mengikuti [Keep a Changelog](https://keepachangelog.com/id/1.1.0/),
versi mengikuti [Semantic Versioning](https://semver.org/lang/id/).

Sumber versi aplikasi: file [`VERSION`](VERSION) (ditampilkan di sidebar desktop dan header mobile).

## [1.0.0] — 2026-08-18

Rilis pertama yang diberi nomor versi.

### Ditambahkan

- Integrasi **Bot KasQ** (Telegram sistem) di samping bot sendiri; Chat ID via `/start`
- **Beberapa foto nota** per transaksi (web, Telegram album, WhatsApp berurutan) + slideshow preview
- **Download ZIP** untuk nota lebih dari satu foto
- Whitelist nomor WhatsApp: hanya nomor terdaftar yang dibalas bot
- Command bot `!link` / `/link` untuk URL laporan publik
- Command `!saldo` / `/saldo` tetap ada; pesan error bot juga menampilkan daftar command
- Format chat tanpa hari: `out#100826#Deskripsi#12000` — hari terisi otomatis dari tanggal
- Salam di dashboard sesuai waktu (Selamat pagi/siang/sore/malam)
- Kompresi foto nota di atas 2 MB (web, Telegram, WhatsApp, import)
- Versioning aplikasi (`KasQ v1.0.0`)

### Diperbaiki

- Preview/download nota 404 (Access Denied / path MinIO)
- Chart dashboard berlatar putih di dark mode
- Tombol Aktifkan WA/Telegram masih bisa diklik padahal sudah ON
- Whitelist WA tidak merespons: WhatsApp mengirim LID, bukan nomor HP
- Input `08…` di whitelist dinormalisasi ke `62…`

### Integrasi bot (ringkas)

```
out#Senin#100826#Deskripsi#12000#Keterangan
out#100826#Deskripsi#12000
!saldo
!link
```
