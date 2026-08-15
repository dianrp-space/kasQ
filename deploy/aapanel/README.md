# Deploy KasQ di aaPanel

File template untuk production deploy. Panduan lengkap: [README.md — Production](../../README.md#production--deploy-di-aapanel-nginx--systemd).

| File | Fungsi |
|------|--------|
| `kasq-backend.service` | systemd unit — Go API (:8084) |
| `ecosystem.config.cjs` | PM2 config frontend (:3008) — **disarankan** |
| `kasq-frontend.service` | systemd frontend (legacy, jika tidak pakai PM2) |
| `nginx-kasq.conf` | Snippet Nginx (tempel di aaPanel site config) |
| `backend.env.production.example` | Template `backend/.env` |
| `frontend.env.production.example` | Template `frontend/.env.production` |
| `../deploy.sh` + `deploy.env.example` | Deploy otomatis (build lokal → rsync) |

## Quick install

```bash
# 1. Build (dari root project) — atau skip jika pakai make deploy
cd backend && go build -o kasq-server ./cmd/server && cd ..
cd frontend && npm ci && npm run build && cd ..

# 2. Env
cp deploy/aapanel/backend.env.production.example backend/.env
cp deploy/aapanel/frontend.env.production.example frontend/.env.production
# edit backend/.env

# 3. Backend systemd
cp deploy/aapanel/kasq-backend.service /etc/systemd/system/
systemctl daemon-reload && systemctl enable --now kasq-backend

# 4. Frontend PM2 (user NVM yang sudah ada, mis. dianrp)
# Matikan systemd frontend jika pernah dipasang:
# systemctl disable --now kasq-frontend 2>/dev/null || true

su - dianrp   # ganti dengan user PM2 kamu
cd /www/wwwroot/kasq.example.com/frontend
pm2 start ecosystem.config.cjs
pm2 save
pm2 startup     # ikuti instruksi (sekali saja)
exit

# 5. deploy.env — set DEPLOY_FRONTEND=pm2 dan DEPLOY_PM2_USER=dianrp

# 6. Nginx — tempel nginx-kasq.conf di aaPanel → Website → Config
nginx -t && nginx -s reload
```

## PM2 perintah berguna

```bash
pm2 status kasq-frontend
pm2 logs kasq-frontend
pm2 restart kasq-frontend
```

Setelah `make deploy`, PM2 di-reload otomatis via `pm2 startOrReload ecosystem.config.cjs`.
