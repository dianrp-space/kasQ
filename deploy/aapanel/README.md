# Deploy KasQ di aaPanel

File template untuk production deploy. Panduan lengkap: [README.md — Production](../../README.md#production--deploy-di-aapanel-nginx--systemd).

| File | Fungsi |
|------|--------|
| `kasq-backend.service` | systemd unit — Go API (:8084) |
| `kasq-frontend.service` | systemd unit — SvelteKit Node (:3008) |
| `nginx-kasq.conf` | Snippet Nginx (tempel di aaPanel site config) |
| `backend.env.production.example` | Template `backend/.env` |
| `frontend.env.production.example` | Template `frontend/.env.production` |
| `../deploy.sh` + `deploy.env.example` | Deploy otomatis (build lokal → rsync) |

## Quick install

Lihat isi lengkap unit file di [README — systemd](../../README.md#6-systemd--backend--frontend).

```bash
# 1. Build (dari root project)
cd backend && go build -o kasq-server ./cmd/server && cd ..
cd frontend && npm ci && npm run build && cd ..

# 2. Env
cp deploy/aapanel/backend.env.production.example backend/.env
cp deploy/aapanel/frontend.env.production.example frontend/.env.production
# edit backend/.env

# 3. systemd (sesuaikan path domain dulu)
cp deploy/aapanel/kasq-backend.service /etc/systemd/system/
cp deploy/aapanel/kasq-frontend.service /etc/systemd/system/
systemctl daemon-reload && systemctl enable --now kasq-backend kasq-frontend

# 4. Nginx — tempel nginx-kasq.conf di aaPanel → Website → Config
nginx -t && nginx -s reload
```
