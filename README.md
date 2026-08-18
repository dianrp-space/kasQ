<p align="center">
  <img src="frontend/src/lib/assets/kasQ-removebg.webp" alt="KasQ Logo" width="140" />
</p>

# KasQ — Kas Ku

Aplikasi fullstack pencatatan pemasukan & pengeluaran kas tim operasional. Bisa input laporan dengan 3 mode, via web, WhatsApp, dan Telegram.

**Stack:** SvelteKit + Go (Gin) + PostgreSQL + MinIO + Whatsmeow + Telebot

**Environment:** Development & deployment di **WSL/Linux** (Ubuntu/Debian).

**Versi:** lihat [`VERSION`](VERSION) · [`CHANGELOG.md`](CHANGELOG.md)

**Repository:** [github.com/dianrp-space/kasQ](https://github.com/dianrp-space/kasQ)

## Clone

Jalankan di **WSL** (SSH key GitHub harus terdaftar di WSL, bukan PowerShell):

```bash
# HTTPS
git clone https://github.com/dianrp-space/kasQ.git
cd kasQ

# atau SSH
git clone git@github.com:dianrp-space/kasQ.git
cd kasQ
```


## Fitur

- Multi-kas per tim dengan saldo terhitung otomatis
- Input via **Web**, **WhatsApp**, atau **Telegram** (bot sistem KasQ atau bot sendiri)
- Link laporan publik untuk admin finance (tanpa login)
- Upload foto nota ke MinIO (view & download)
- Role: admin (kelola tim/user) & ops (input transaksi)

## Prasyarat (WSL)

Semua tool diinstall **di dalam WSL**, bukan Windows:

| Tool | Versi | Install (Ubuntu/Debian) |
|------|-------|-------------------------|
| Go | 1.23+ | [go.dev/dl](https://go.dev/dl/) atau `sudo snap install go --classic` |
| Node.js | 20+ | `curl -fsSL https://deb.nodesource.com/setup_20.x \| sudo -E bash - && sudo apt install -y nodejs` |
| PostgreSQL | 16+ | `sudo apt update && sudo apt install -y postgresql postgresql-contrib` |
| MinIO | latest | Via `make minio` (auto-download binary) |
| Make | any | `sudo apt install -y make` |

### Tips WSL

- Simpan project di filesystem Linux (`~/projects/kasQ`) untuk performa Go/npm lebih baik.
- Jika project ada di `/mnt/e/...`, tetap bisa jalan tapi build lebih lambat.
- Akses dari browser Windows: `http://localhost:3008` (WSL2 forward port otomatis).

## Quick Start

```bash
# 1. Clone repo (di WSL)
git clone https://github.com/dianrp-space/kasQ.git ~/projects/kasQ
cd ~/projects/kasQ

# 2. Install PostgreSQL & start service
sudo apt update && sudo apt install -y postgresql postgresql-contrib
sudo service postgresql start

# 3. Setup database
make setup-db

# 4. Copy env & install dependencies
make setup

# 5. Terminal terpisah — jalankan MinIO
make minio
# Buat bucket "kasq" di http://localhost:9001

# 6. Jalankan app (backend + frontend)
make dev
```

Buka http://localhost:3008

## Setup Manual

### PostgreSQL

Jika sudah buat database manual (misal user `postgres`):

```env
# backend/.env
DATABASE_URL=postgres://postgres:PASSWORD@localhost:5432/kasq?sslmode=disable
```

Verifikasi koneksi:

```bash
make fix-crlf    # jalankan sekali jika error "pipefail" di WSL
make setup-db    # cek koneksi, buat DB jika belum ada
```

Atau buat database manual:

```bash
sudo -u postgres psql -c "CREATE DATABASE kasq;"
```

### MinIO

```bash
make minio
```

Console: http://localhost:9001 — login `minioadmin` / `minioadmin`  
Buat bucket **`kasq`**.

Struktur object di bucket:

| Path MinIO | Isi |
|------------|-----|
| `nota/{team-id}/{YYYY/MM}/{uuid}.ext` | Foto nota transaksi |
| `branding/logo.ext` | Logo aplikasi |
| `branding/favicon.ext` | Favicon |
| `avatars/{user-id}.ext` | Foto profil user |

### Environment

```bash
make setup-env
# Edit backend/.env jika perlu
```

`backend/.env.example`:

```env
DATABASE_URL=postgres://postgres:YOUR_PASSWORD@localhost:5432/kasq?sslmode=disable
MINIO_ENDPOINT=s3.example.com
MINIO_ACCESS_KEY=your-access-key
MINIO_SECRET_KEY=your-secret-key
MINIO_BUCKET=kasq
MINIO_USE_SSL=true
JWT_SECRET=change-me
APP_URL=http://localhost:3008
API_URL=http://localhost:8084
PORT=8084
```

Bot Telegram sistem (opsional). Jika diisi, user bisa pilih **Bot KasQ** di halaman Integrasi tanpa membuat bot sendiri:

```env
TELEGRAM_BOT_TOKEN=123456:ABC-token-dari-BotFather
```

> **MinIO remote:** bisa pakai `MINIO_ENDPOINT=https://s3.example.com` — SSL otomatis terdeteksi.

> **Port bentrok?** Ubah `PORT` di `backend/.env` — vite proxy otomatis mengikuti via `BACKEND_PORT`.

## Development

**Opsi A — satu perintah (backend + frontend):**

```bash
make dev
```

**Opsi B — dua terminal terpisah:**

```bash
# Terminal 1
make dev-backend

# Terminal 2
make dev-frontend
```

**Terminal 3 — MinIO (jika belum jalan):**

```bash
make minio
```

## Auth & Email

Register, verifikasi email, dan forgot password membutuhkan SMTP. Tambahkan di `backend/.env`:

```env
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password
SMTP_FROM=KasQ <noreply@kasq.local>
```

Flow:
- **Register** → email konfirmasi → klik link → baru bisa login
- **Forgot password** → email reset link → set password baru
- User dibuat admin via panel admin langsung aktif (tanpa verifikasi email)

## Login Seed (development)

| Email | Password | Role |
|-------|----------|------|
| admin@kasq.local | admin123 | Admin |
| ops@kasq.local | admin123 | Ops (Tim Batam) |

## Format Input WA/Telegram

```
out#Senin#100826#Beli air minum isi ulang galon#12000
out#100826#Beli air minum isi ulang galon#12000
in#Sabtu#010826#Refill kas Batam#2000000
```

- Hari opsional — kalau dikosongkan, terisi otomatis dari tanggal (`out#100826#...`).
- Pengeluaran dengan nota: kirim **1–10 foto**. Telegram: caption di foto mana pun (album OK). WhatsApp: beberapa foto berurutan, caption di foto terakhir (atau salah satu).
- Cek saldo: `/saldo` (Telegram) atau `!saldo` (WA)
- Link laporan: `/link` atau `!link`

### Bot Telegram

Dua opsi di halaman **Integrasi**:

1. **Bot KasQ** — bot permanen sistem (`TELEGRAM_BOT_TOKEN` di `backend/.env`). User buka bot, kirim `/start`, salin Chat ID ke form, lalu aktifkan. Hanya chat pribadi.
2. **Bot sendiri** — token dari [@BotFather](https://t.me/BotFather), boleh grup atau private.

## Production — Deploy di aaPanel (Nginx + systemd)

Panduan ini untuk deploy KasQ di VPS Linux dengan **aaPanel**, **Nginx** sebagai reverse proxy, dan **systemd** untuk backend (Go). Frontend SvelteKit juga dijalankan via systemd (adapter Node).

Template siap pakai ada di folder [`deploy/aapanel/`](deploy/aapanel/).

### Ringkasan arsitektur

```
Browser → Nginx (443) → /api/*  → kasq-backend  (:8084, systemd)
                      → /*      → kasq-frontend (:3008, PM2)
PostgreSQL (:5432) + MinIO/S3 (remote)
```

### Deploy otomatis — build lokal, server tanpa Go/npm

**Bisa.** Server cukup terima hasil build; tidak perlu `go build` atau `npm run build` di VPS.

Alur:

1. **Sekali** setup server: PostgreSQL, Nginx, systemd, `backend/.env`, Node.js
2. **Setiap update** dari laptop/WSL: `make deploy` → build lokal → rsync artefak → restart service

Yang di-upload otomatis:

| Path di server | Isi |
|----------------|-----|
| `backend/kasq-server` | Binary Go (linux amd64, cross-compile) |
| `frontend/build/` | Hasil `npm run build` (SvelteKit adapter-node) |

**Tidak** di-upload / **tetap di server**:

| Path | Alasan |
|------|--------|
| `backend/.env` | Secret production — buat manual di server |
| `backend/data/wa-sessions/` | Session WA — jangan ditimpa |
| Source code, `node_modules`, Go module | Tidak dibutuhkan di production |

Setup deploy (sekali, di mesin dev/WSL):

```bash
cp deploy/deploy.env.example deploy/deploy.env
nano deploy/deploy.env   # DEPLOY_HOST, DEPLOY_USER, DEPLOY_PATH, SSH key
chmod +x deploy/deploy.sh
```

Deploy / update:

```bash
make deploy
# atau: ./deploy/deploy.sh
```

Script akan: build backend (`GOOS=linux`) → build frontend → `rsync` ke server → restart backend (systemd) + frontend (PM2).

> **Syarat mesin dev:** Go, Node.js, `rsync`, akses SSH ke VPS (WSL/Linux).  
> **Syarat server:** Node.js (jalankan `frontend/build/index.js`), **tidak perlu Go**.

### 1. Prasyarat di server

Via **aaPanel → App Store**, pastikan terinstall:

| Komponen | Catatan |
|----------|---------|
| **Nginx** | Reverse proxy (biasanya sudah ada) |
| **PostgreSQL** | v14+ via aaPanel, atau database eksternal |
| **Node.js** | v20+ (App Store → Node version manager) — untuk jalankan frontend build |
| **Go** | Hanya jika build di server; **tidak perlu** jika pakai `make deploy` |

<details>
<summary>Install Go di server (opsional — skip jika pakai deploy otomatis)</summary>

Install Go (SSH root):

```bash
wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
source /etc/profile
go version
```

</details>

MinIO bisa dipasang terpisah, atau pakai S3-compatible yang sudah ada. Buat bucket **`kasq`**.

### 2. Setup awal server (sekali)

Buat folder deploy & env production (via SSH):

```bash
mkdir -p /www/wwwroot/kasq.example.com/backend/data/wa-sessions
nano /www/wwwroot/kasq.example.com/backend/.env   # lihat deploy/aapanel/backend.env.production.example
```

Install systemd + Nginx (lihat §6 dan §7). Setelah itu, update berikutnya cukup `make deploy` dari mesin dev.

<details>
<summary>Alternatif: clone full repo & build di server</summary>

```bash
mkdir -p /www/wwwroot/kasq.example.com
cd /www/wwwroot/kasq.example.com
git clone https://github.com/dianrp-space/kasQ.git .
# atau: git clone git@github.com:dianrp-space/kasQ.git .
```

Lalu lanjut build manual §5.

</details>

### 3. PostgreSQL (aaPanel)

1. **App Store → PostgreSQL** → install & catat password
2. **Database → Add database**
   - Name: `kasq`
   - User: `kasq` (atau pakai user postgres)
   - Password: buat password kuat
3. Catat connection string untuk `.env`:

```env
DATABASE_URL=postgres://kasq:PASSWORD@127.0.0.1:5432/kasq?sslmode=disable
```

> Migrasi database jalan otomatis saat backend pertama kali start.

### 4. Environment production

**Backend** — salin template:

```bash
cp deploy/aapanel/backend.env.production.example backend/.env
nano backend/.env
```

Isi penting:

| Variabel | Contoh production |
|----------|-------------------|
| `APP_URL` | `https://kasq.example.com` |
| `API_URL` | `https://kasq.example.com` |
| `PORT` | `8084` (internal, tidak perlu dibuka publik) |
| `JWT_SECRET` | string random panjang |
| `MINIO_*` | endpoint & kredensial S3/MinIO |

**Frontend** — sebelum build:

```bash
cp deploy/aapanel/frontend.env.production.example frontend/.env.production
```

Biarkan `PUBLIC_API_URL` **kosong** supaya browser memanggil `/api/...` lewat domain yang sama (Nginx yang proxy ke backend).

### 5. Build aplikasi

> **Pakai `make deploy`?** Lewati section ini — build jalan otomatis di mesin dev.

```bash
cd /www/wwwroot/kasq.example.com

# Backend
cd backend
go mod tidy
go build -o kasq-server ./cmd/server

# Frontend
cd ../frontend
npm ci
npm run build
```

Pastikan folder `backend/data/wa-sessions/` bisa ditulis (session WhatsApp):

```bash
mkdir -p backend/data/wa-sessions
chown -R www:www backend/data
```

### 6. systemd (backend) + PM2 (frontend)

Template backend: [`deploy/aapanel/kasq-backend.service`](deploy/aapanel/kasq-backend.service)  
Template frontend PM2: [`deploy/aapanel/ecosystem.config.cjs`](deploy/aapanel/ecosystem.config.cjs)

**Ganti semua** `kasq.example.com` dengan domain/path kamu (mis. `kasq.dianrp.com`).

> Frontend pakai **PM2** dengan user NVM yang sudah ada (mis. `dianrp`) — tidak perlu install Node system-wide.  
> Matikan systemd frontend jika pernah dipasang: `systemctl disable --now kasq-frontend`

#### Backend — `/etc/systemd/system/kasq-backend.service`

```ini
[Unit]
Description=KasQ Backend (Go API + Bot WA/Telegram)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=www
Group=www
WorkingDirectory=/www/wwwroot/kasq.example.com/backend
EnvironmentFile=/www/wwwroot/kasq.example.com/backend/.env
ExecStart=/www/wwwroot/kasq.example.com/backend/kasq-server
Restart=on-failure
RestartSec=5
LimitNOFILE=65535
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

| Field | Fungsi |
|-------|--------|
| `WorkingDirectory` | Folder `backend/` — WA session disimpan di `data/wa-sessions/` relatif ke sini |
| `EnvironmentFile` | Load `backend/.env` (PORT, DATABASE_URL, MinIO, JWT, SMTP) |
| `ExecStart` | Binary hasil `go build -o kasq-server` |
| `User=www` | User aaPanel (sesuaikan jika berbeda) |

> **PostgreSQL dari aaPanel** (`/www/server/pgsql/bin/psql`): path `psql` **tidak dipakai** systemd/KasQ backend. Backend connect via `DATABASE_URL` (TCP ke `127.0.0.1:5432`). Unit `postgresql.service` biasanya **tidak ada** di aaPanel — aman pakai `After=network-online.target` saja. Cek nama unit PG: `systemctl list-units --type=service | grep -i pg`

Port backend dari `.env` → `PORT=8084` (internal, di-proxy Nginx).

#### Frontend — PM2 (`ecosystem.config.cjs`)

Port **3008**, di-proxy Nginx ke `/`. File config di-upload otomatis ke `frontend/ecosystem.config.cjs` saat `make deploy`.

**Setup sekali** (sebagai user PM2 / NVM, mis. `dianrp`):

```bash
cd /www/wwwroot/kasq.example.com/frontend
pm2 start ecosystem.config.cjs
pm2 save
pm2 startup    # ikuti instruksi sekali
```

Set `DEPLOY_FRONTEND=pm2` dan `DEPLOY_PM2_USER=dianrp` di `deploy/deploy.env`.

```bash
pm2 status kasq-frontend
pm2 logs kasq-frontend
pm2 restart kasq-frontend   # atau cukup make deploy (auto reload)
```

<details>
<summary>Alternatif: systemd frontend (legacy)</summary>

Template: [`deploy/aapanel/kasq-frontend.service`](deploy/aapanel/kasq-frontend.service) — butuh Node di path system (`/usr/bin/node`). Set `DEPLOY_FRONTEND=systemd` di deploy.env.

</details>

#### Install backend & jalankan

```bash
cd /www/wwwroot/kasq.example.com

nano deploy/aapanel/kasq-backend.service
cp deploy/aapanel/kasq-backend.service /etc/systemd/system/

systemctl daemon-reload
systemctl enable --now kasq-backend
systemctl status kasq-backend
```

Perintah berguna:

```bash
journalctl -u kasq-backend -f    # log backend
systemctl restart kasq-backend   # setelah update backend
pm2 logs kasq-frontend           # log frontend (user PM2)
```

**Update deploy backend:**

```bash
cd /www/wwwroot/kasq.example.com/backend
git pull
go build -o kasq-server ./cmd/server
systemctl restart kasq-backend
```

**Update deploy frontend:** `make deploy` (PM2 reload otomatis) atau manual:

```bash
cd /www/wwwroot/kasq.example.com/frontend
npm ci && npm run build
pm2 reload kasq-frontend
```

### 7. Nginx di aaPanel

1. **Website → Add site** → domain `kasq.example.com`
2. **SSL → Let's Encrypt** → aktifkan HTTPS
3. **Website → kasq.example.com → Config**
4. Tempel isi [`deploy/aapanel/nginx-kasq.conf`](deploy/aapanel/nginx-kasq.conf) **di dalam** block `server { ... }`, **sebelum** baris penutup `}`

Atau lewat **Reverse proxy** aaPanel (alternatif):

| Path | Target |
|------|--------|
| `/api` | `http://127.0.0.1:8084` |
| `/` | `http://127.0.0.1:3008` |

Pastikan **client_max_body_size** minimal `10m` (upload nota/avatar).

5. Reload Nginx:

```bash
nginx -t && nginx -s reload
```

### 8. Firewall & port

Port **8084** dan **3008** cukup bind ke `127.0.0.1` — **jangan** buka ke publik. Yang diakses internet hanya **80/443** via Nginx.

Di aaPanel **Security**, buka port 80, 443, dan port SSH saja.

### 9. Checklist setelah deploy

- [ ] `https://kasq.example.com` — halaman login tampil
- [ ] Login seed / user admin berhasil
- [ ] Upload nota & avatar jalan (MinIO bucket `kasq`)
- [ ] Email verifikasi terkirim (SMTP benar)
- [ ] Integrasi WA/Telegram (jika dipakai) — scan QR dari halaman Integrasi
- [ ] `systemctl status kasq-backend` → `active (running)`
- [ ] `curl -s http://127.0.0.1:8084/api/health` → `{"status":"ok",...}`

### 10. Troubleshooting

| Gejala | Solusi |
|--------|--------|
| Halaman "Memuat..." + 404 `/_app/*.js` | aaPanel regex static `*.js` menang — tambah `location ^~ /_app/` (lihat `nginx-kasq.conf`) atau comment block static js/css aaPanel |
| 502 Bad Gateway | Cek `systemctl status kasq-backend` + `pm2 status kasq-frontend`, pastikan port 8084 & 3008 listen |
| API 401 / cookie gagal | Pastikan `APP_URL` pakai `https://` yang sama dengan domain |
| Upload gagal | Cek `client_max_body_size` Nginx + kredensial MinIO |
| MinIO Access Denied | Pastikan IAM/key punya `PutObject` + `GetObject` pada bucket `kasq` |
| WA disconnect setelah restart | Normal jika session ada — scan ulang QR; folder `backend/data/wa-sessions` harus persisten |

---

## Production (generic Linux, tanpa Docker)

Jika tidak pakai aaPanel, lihat template yang sama di [`deploy/aapanel/`](deploy/aapanel/) — langkah build & systemd serupa.

### Backend

```bash
cd backend
go build -o kasq-server ./cmd/server
./kasq-server
```

### Frontend

```bash
cd frontend
npm run build
PORT=3008 node build/index.js
```

### Nginx reverse proxy

Lihat [`deploy/aapanel/nginx-kasq.conf`](deploy/aapanel/nginx-kasq.conf).

## Struktur Project

```
kasQ/
├── backend/          # Go API + bot WA/Tele
├── frontend/         # SvelteKit UI
├── deploy/aapanel/   # Template systemd + nginx + .env production
├── Makefile          # Perintah dev/build/setup
└── README.md
```

## Makefile Commands

| Command | Deskripsi |
|---------|-----------|
| `make setup` | Copy .env + install deps |
| `make setup-db` | Buat user/database PostgreSQL |
| `make deps` | go mod tidy + npm install |
| `make dev` | Backend + frontend sekaligus |
| `make dev-backend` | Go server saja |
| `make dev-frontend` | SvelteKit dev saja |
| `make minio` | Start MinIO lokal |
| `make test` | Backend unit tests |
| `make build` | Build production binaries |
| `make deploy` | Build lokal + rsync artefak ke server (butuh `deploy/deploy.env`) |

## API Endpoints

- `POST /api/auth/login` — Login
- `GET /api/teams/:id/transactions` — List transaksi
- `POST /api/teams/:id/transactions` — Input transaksi (multipart)
- `GET /api/public/report/:token` — Laporan publik finance
- `PUT /api/teams/:id/integrations/wa` — Aktifkan/nonaktifkan WA
- `PUT /api/teams/:id/integrations/tele` — Setup bot Telegram
