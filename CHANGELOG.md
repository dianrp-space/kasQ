# Changelog

Semua perubahan penting KasQ dicatat di sini.

Format mengikuti [Keep a Changelog](https://keepachangelog.com/id/1.1.0/),
versi mengikuti [Semantic Versioning](https://semver.org/lang/id/).

Sumber versi aplikasi: file [`VERSION`](VERSION) (ditampilkan di sidebar desktop dan header mobile; klik untuk membuka changelog).

## [1.0.4] — 2026-08-20

### Ditambahkan

- **Export Excel dan PDF** di dashboard dan laporan publik (mengikuti filter bulan, jenis, dan pencarian)
- Ikon sumber transaksi (WhatsApp, Telegram, Web) di dashboard dan laporan publik
- Nama periode pada kartu saldo, misalnya *Saldo awal periode Agustus 2026*
- Menu foto profil: lihat atau ubah foto (tanpa lewat menu Profil)
- Toggle tema, avatar, dan tombol **Logout** di bagian bawah sidebar desktop

### Diubah

- Label jenis: **Masuk** / **Keluar** (sebelumnya Pemasukan / Pengeluaran)
- Kartu Link laporan dan Command bot di Integrasi bersandingan di desktop
- Tombol Import Excel tidak terdorong ke tepi layar
- Placeholder field lebih redup
- Logout: outline + latar merah; di mobile hanya ikon

### Diperbaiki

- Teks deskripsi/keterangan panjang di PDF tidak lagi terpotong — wrap ke baris berikutnya
- Animasi baris saat menyusun ulang transaksi di tanggal yang sama

## [1.0.3] — 2026-08-19

### Ditambahkan

- Halaman **Changelog** (`/changelog`) — klik nomor versi di sidebar, header, atau halaman masuk
- Poles UI halaman masuk, profil, input transaksi, integrasi, support, dan dashboard (panel, tab, dan tombol yang lebih konsisten)
- Filter dashboard **langsung diterapkan** saat bulan atau jenis dipilih (tanpa tombol Terapkan)
- Nominal transaksi berwarna: pemasukan hijau, pengeluaran merah
- Kolom nota di antara jenis dan deskripsi; aksi edit/hapus jadi ikon saja

### Diperbaiki

- Tab jenis/telegram kurang kontras saat tidak dipilih
- Ikon mata pada field kata sandi (bukan emoji)
- Crash Tailwind `primary-950` pada zona unggah nota

## [1.0.2] — 2026-08-19

### Ditambahkan

- **Susun ulang transaksi** di dashboard (seret di desktop, panah di ponsel) untuk mengoreksi urutan di tanggal yang sama
- **Pencarian** di tabel dashboard (deskripsi, keterangan, total, tanggal, jenis, sumber)
- Opsi **per halaman** 20 / 50 / 100 / 200 pada pagination dashboard

### Diperbaiki

- Ikon dropdown “Semua jenis” menimpa teks
- Tombol Terapkan tanpa warna latar yang jelas

## [1.0.1] — 2026-08-18

### Ditambahkan

- Bot WhatsApp mendukung **self-chat**: login nomor pribadi, kirim perintah ke diri sendiri (Pesan tersimpan / nomor sendiri)
- Teks bantuan di Integrasi WA: whitelist nomor bot sendiri agar chat dari orang lain tidak dibalas

### Diperbaiki

- Pesan `IsFromMe` di self-chat sebelumnya diabaikan, jadi chat ke nomor sendiri tidak dijawab

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
