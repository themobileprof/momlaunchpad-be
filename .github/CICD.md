# GitHub Actions CI/CD Documentation

This project uses GitHub Actions for continuous integration and deployment.

## Workflows

### 1. CI (`.github/workflows/ci.yml`)

**Triggers:**
- Push to `main` or `develop`
- Pull requests to `main` or `develop`

(Version tags do **not** re-run CI — only the deploy workflow runs on `v*` tags.)

**Jobs (parallel, then build):**
- **Test** — PostgreSQL + Redis, `go test`, Codecov
- **Lint** — golangci-lint
- **Format** — `gofmt` + `go vet`
- **Security** — Gosec (SARIF to GitHub Security)
- **Build** — runs after test/lint/format pass
  - Multi-platform image (amd64, arm64)
  - **Push** only on `main` or `v*` tags (not on `develop` or PRs)
  - **Trivy** scan when an image is pushed
  - PRs and `develop` still build the image to verify the Dockerfile

**Image tags (when pushed):** `latest`, semver (`0.1.1`, `0.1`), `sha-abc1234`, branch/PR refs

---

### 2. Deploy to Production (`.github/workflows/deploy.yml`)

**Triggers:**
- Version tags (`v*`) only (not full CI)
- Manual workflow dispatch (with environment selection)

**Image tag on release:** Tries `0.1.4` (no `v`), then `sha-<commit>` from the last `main` CI build, then `latest`. Tag releases after CI has passed on `main` for that commit.

**Host port:** MomLaunchpad listens on **`8083`** on the server (`PORT=8083` is set at deploy time). The container uses **`--network host`** so `DATABASE_URL=...@localhost:5432/...` works when Postgres only binds to `127.0.0.1` (typical on DigitalOcean/LAMP VPS). Nginx for MomLaunchpad: `deploy/nginx/api.momlaunchpad.com.conf` → `127.0.0.1:8083`. **Do not** use `api.themobileprof.com.conf` for MomLaunchpad — that vhost is for legacy `themobileprof-backend` on port **3000**.

**Database/Redis URLs:** In `ENV_FILE`, use **`localhost`** (or `127.0.0.1`) for Postgres/Redis on the same VM — not `172.17.0.1` or `host.docker.internal`. Deploy uses host networking specifically because bridge access to Postgres often times out unless you reconfigure `listen_addresses` and `pg_hba.conf`.

**Deployment Steps:**
1. SSH into production/staging server
2. Write `ENV_FILE` secret to a temporary env file on the server
3. Pull Docker image from Docker Hub (semver tag → `sha-*` → `latest`)
4. Stop and replace **only** `momlaunchpad-api` (other containers untouched)
5. Start new container with **`--network host`**, `PORT=8083`, and persistent uploads volume
6. Verify `NetworkMode=host` and `/health` on port 8083
7. Run database migrations on the server (from the deployed image’s `/app/migrations`)
8. Clean up env file and old images

**Environments:**
- Production
- Staging

**Rollbacks:** run `migrate down` manually on the server (not via GitHub Actions).

---

## Required GitHub Secrets

### Repository Secrets

```bash
# SSH Access
SSH_HOST=your-server-ip-or-domain
SSH_USERNAME=deploy
SSH_PRIVATE_KEY=your-ssh-private-key

# Staging (optional)
STAGING_SSH_HOST=staging-server-ip
STAGING_SSH_USERNAME=deploy

# DockerHub
DOCKERHUB_USERNAME=your-dockerhub-username
DOCKERHUB_TOKEN=your-dockerhub-access-token

# Environment Variables (RECOMMENDED: Single Secret)
# Copy your entire .env file and save as ENV_FILE secret
ENV_FILE=<paste your entire production .env file here>
# This should include:
# - DATABASE_URL=postgresql://user:pass@localhost:5432/momlaunchpad?sslmode=disable
# - REDIS_URL=redis://localhost:6379/0  (if enabled)
# - Do NOT use host.docker.internal or 172.17.0.1 — deploy uses --network host
# - PORT and UPLOAD_DIR are set by the deploy workflow; omit or they will be overwritten
# - DEEPSEEK_API_KEY
# - JWT_SECRET
# - All other environment variables from .env.example

```

### Setting Secrets

**Recommended: Copy entire .env file as ENV_FILE**

```bash
# Via GitHub CLI (copy entire .env file)
gh secret set ENV_FILE < .env.production

# Or via GitHub Web UI:
# 1. Copy your entire production .env file
# 2. Go to Settings → Secrets and variables → Actions
# 3. New repository secret: ENV_FILE
# 4. Paste entire .env contents

# Set other required secrets
gh secret set SSH_HOST -b"1.2.3.4"
gh secret set SSH_USERNAME -b"deploy"
gh secret set SSH_PRIVATE_KEY < ~/.ssh/deploy_key
gh secret set DOCKERHUB_USERNAME -b"your-dockerhub-username"
gh secret set DOCKERHUB_TOKEN -b"your-dockerhub-token"

# Via GitHub Web UI
# Settings → Secrets and variables → Actions → New repository secret
```

---

## Deployment Process

### Automatic Deployment (Tag-based)

```bash
# 1. Create and push version tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 2. GitHub Actions automatically:
#    - Builds Docker image
#    - Runs security scan
#    - Deploys to production
#    - Runs migrations
#    - Verifies health check
```

### Manual Deployment

```bash
# Via GitHub CLI
gh workflow run deploy.yml \
  --ref main \
  -f environment=staging

# Via GitHub Web UI
# Actions → Deploy to Production → Run workflow
# Select environment (production/staging)
```

---

## Server Setup

### 1. Provision Server

```bash
# Create VM (example: DigitalOcean, AWS EC2, etc.)
# Ubuntu 22.04 LTS recommended
# Minimum: 2 CPU, 2GB RAM, 20GB disk
```

### 2. Install Dependencies

```bash
# SSH into server
ssh deploy@your-server

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh
sudo usermod -aG docker $USER

# Install Docker Compose (optional, for multi-container setups)
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Verify installations
docker --version
docker-compose --version
```

### 3. Setup Deploy User

```bash
# Create deploy user
sudo adduser deploy
sudo usermod -aG docker deploy

# Setup SSH key for GitHub Actions
sudo su - deploy
mkdir -p ~/.ssh
chmod 700 ~/.ssh

# Add your GitHub Actions public key to authorized_keys
echo "ssh-rsa AAAA..." >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
```

### 4. Setup External Services

#### PostgreSQL (Managed Database Recommended)

```bash
# Option 1: Managed (DigitalOcean, AWS RDS, etc.)
# Create database: momlaunchpad
# Get connection string

# Option 2: Self-hosted with Docker
docker run -d \
  --name postgres \
  --restart unless-stopped \
  -e POSTGRES_DB=momlaunchpad \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=secure_password \
  -v postgres_data:/var/lib/postgresql/data \
  -p 5432:5432 \
  postgres:15-alpine
```

#### Redis (Optional)

```bash
# Option 1: Managed (Redis Cloud, AWS ElastiCache)
# Get connection string

# Option 2: Self-hosted with Docker
docker run -d \
  --name redis \
  --restart unless-stopped \
  -v redis_data:/data \
  -p 6379:6379 \
  redis:7-alpine
```

### 5. Setup Nginx (Reverse Proxy)

```bash
# Install Nginx
sudo apt update
sudo apt install nginx

# Create config
sudo nano /etc/nginx/sites-available/momlaunchpad
```

```nginx
upstream backend {
    server localhost:8080;
    keepalive 32;
}

server {
    listen 80;
    server_name api.momlaunchpad.com;
    
    # Redirect HTTP to HTTPS
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.momlaunchpad.com;
    
    # SSL certificates (Let's Encrypt)
    ssl_certificate /etc/letsencrypt/live/api.momlaunchpad.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.momlaunchpad.com/privkey.pem;
    
    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    
    # Proxy settings
    location / {
        proxy_pass http://backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket support
        proxy_read_timeout 86400;
        proxy_send_timeout 86400;
    }
    
    # Health check endpoint (bypass rate limiting)
    location /health {
        proxy_pass http://backend;
        access_log off;
    }
}
```

```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/momlaunchpad /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx

# Setup SSL with Let's Encrypt
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d api.momlaunchpad.com
```

### 6. Setup Firewall

```bash
# UFW (Ubuntu)
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw enable

# Only allow internal access to Docker ports
# Backend runs on localhost:8080 (not exposed)
```

---

## Monitoring & Logging

### View Logs

```bash
# Container logs
docker logs -f momlaunchpad-api

# Nginx logs
sudo tail -f /var/log/nginx/access.log
sudo tail -f /var/log/nginx/error.log

# System logs
journalctl -u docker -f
```

### Health Monitoring

```bash
# Setup cron job for health checks
crontab -e
```

```cron
# Check health every 5 minutes
*/5 * * * * curl -f http://localhost:8080/health || systemctl restart momlaunchpad-api
```

### Metrics (Optional)

```bash
# Install Prometheus Node Exporter
docker run -d \
  --name node-exporter \
  --restart unless-stopped \
  -p 9100:9100 \
  prom/node-exporter

# Setup Grafana dashboard
docker run -d \
  --name grafana \
  --restart unless-stopped \
  -p 3000:3000 \
  grafana/grafana
```

---

## Rollback Procedure

### Automatic Rollback (if health check fails)

The deploy workflow automatically checks `/health` endpoint after deployment. If it fails, the workflow exits with error and the old container remains running.

### Manual Rollback

```bash
# SSH into server
ssh deploy@your-server

# List images
docker images | grep momlaunchpad

# Stop current container
docker stop momlaunchpad-api
docker rm momlaunchpad-api

# Start previous version (--network host, same as production deploy)
docker run -d \
  --name momlaunchpad-api \
  --restart unless-stopped \
  --network host \
  -v /var/www/momlaunchpad-api/uploads:/app/uploads \
  --env-file /path/to/production.env \
  -e PORT=8083 \
  -e UPLOAD_DIR=/app/uploads \
  YOUR_DOCKERHUB_USER/momlaunchpad-be:v1.0.0

# Verify
curl http://localhost:8083/health
```

### Database Rollback

Run on the server (not from GitHub Actions):

```bash
migrate -path ./migrations \
  -database "$DATABASE_URL" \
  down 1
```

---

## Troubleshooting

### Deployment Fails

```bash
# Check workflow logs on GitHub
gh run view --log

# SSH into server and check
ssh deploy@your-server
docker ps -a
docker logs momlaunchpad-api
```

### Container Won't Start

```bash
# Check environment variables
docker inspect momlaunchpad-api | grep -A 20 Env

# Check logs
docker logs momlaunchpad-api

# Test database connection
docker run --rm postgres:15-alpine psql "$DATABASE_URL" -c "SELECT 1;"
```

### Health Check Fails

```bash
# Check if container is running
docker ps | grep momlaunchpad

# Check container health
docker inspect momlaunchpad-api | grep -A 10 Health

# Test endpoint
curl -v http://localhost:8080/health

# Check Nginx
sudo nginx -t
sudo systemctl status nginx
```

---

## Best Practices

### 1. Use Semantic Versioning

```bash
# Patch: Bug fixes (v1.0.1)
git tag v1.0.1

# Minor: New features (v1.1.0)
git tag v1.1.0

# Major: Breaking changes (v2.0.0)
git tag v2.0.0
```

### 2. Test in Staging First

```bash
# Deploy to staging
gh workflow run deploy.yml -f environment=staging

# Verify
curl https://staging.momlaunchpad.com/health

# If successful, deploy to production
git tag v1.2.3
git push origin v1.2.3
```

### 3. Database Backups

```bash
# Automated daily backups
crontab -e
```

```cron
# Daily backup at 2 AM
0 2 * * * docker exec postgres pg_dump -U postgres momlaunchpad | gzip > /backups/db-$(date +\%Y\%m\%d).sql.gz

# Cleanup old backups (keep 30 days)
0 3 * * * find /backups -name "db-*.sql.gz" -mtime +30 -delete
```

### 4. Monitoring Alerts

```bash
# Setup Uptime Robot or similar
# Monitor: https://api.momlaunchpad.com/health
# Alert on failure
```

---

## Security Checklist

- [ ] SSH keys used (no password auth)
- [ ] Firewall configured (UFW/iptables)
- [ ] HTTPS/SSL enabled (Let's Encrypt)
- [ ] Secrets stored in GitHub Secrets (not in code)
- [ ] Database uses strong password
- [ ] Database accessible only from backend
- [ ] Rate limiting enabled
- [ ] Docker runs as non-root user
- [ ] Regular security updates
- [ ] Vulnerability scans enabled (Trivy)

---

## Cost Optimization

### 1. Use Managed Services

- **Database:** DigitalOcean Managed PostgreSQL ($15/mo)
- **Redis:** Redis Cloud Free Tier
- **Server:** DigitalOcean Droplet ($12/mo for 2GB RAM)
- **CDN:** Cloudflare Free Tier

### 2. Resource Limits

```bash
# Limit container resources
docker run -d \
  --name momlaunchpad-api \
  --memory="512m" \
  --cpus="1.0" \
  ...
```

### 3. Log Rotation

Already configured in deploy workflow:
```bash
--log-opt max-size=10m \
--log-opt max-file=3
```

---

## Support

For issues:
- Check workflow logs: `gh run view --log`
- View server logs: `docker logs momlaunchpad-api`
- Contact: [Your support channel]
