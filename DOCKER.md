# Docker Development Guide

This guide covers running MomLaunchpad backend with Docker **for local development and testing only**.

**⚠️ For production deployment, see [.github/CICD.md](.github/CICD.md) - GitHub Actions handles production deployments.**

## Purpose

- **Development:** Local testing with hot reload
- **Testing:** CI/CD test environments
- **NOT for production:** Use GitHub Actions for production deployments

## Prerequisites

- Docker 20.10+
- Docker Compose 2.0+
- DeepSeek API key (for testing)

## Quick Start

### 1. Setup Environment Variables

```bash
# Copy example environment file
cp .env.docker .env

# Edit .env with your actual values
nano .env
```

**Required variables:**
```env
DEEPSEEK_API_KEY=sk-your-actual-key-here
JWT_SECRET=your-super-secret-key-min-32-chars
DB_PASSWORD=secure_postgres_password
```

### 2. Start All Services

```bash
# Start in detached mode
docker-compose up -d

# View logs
docker-compose logs -f backend

# Check status
docker-compose ps
```

### 3. Run Database Migrations

```bash
# Run migrations inside the container
docker-compose exec backend ./server migrate up

# Or manually with psql
docker-compose exec postgres psql -U postgres -d momlaunchpad -f /docker-entrypoint-initdb.d/001_init_schema.sql
```

### 4. Test the API

```bash
# Health check
curl http://localhost:8080/health

# Register user
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "name": "Test User",
    "language": "en"
  }'
```

## Docker Compose Services

### Architecture

```
┌─────────────────────────────────────────────┐
│               Load Balancer                  │
│            (Nginx/Traefik)                   │
└──────────────────┬──────────────────────────┘
                   │
┌──────────────────▼──────────────────────────┐
│          Backend API (Port 8080)             │
│   - WebSocket: /ws/chat                      │
│   - HTTP: /api/*                             │
└─────┬──────────────────────────┬─────────────┘
      │                          │
┌─────▼──────┐          ┌────────▼─────┐
│ PostgreSQL │          │     Redis    │
│ (Port 5432)│          │  (Port 6379) │
└────────────┘          └──────────────┘
```

### Services

1. **backend** - Go API server
   - Port: 8080
   - Health: `/health`
   - Dependencies: PostgreSQL, Redis

2. **postgres** - PostgreSQL 15
   - Port: 5432
   - Database: `momlaunchpad`
   - User: `postgres`

3. **redis** - Redis 7 (optional cache)
   - Port: 6379

## Development Setup

### Hot Reload Development

```bash
# Use development override
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up

# This mounts your source code for live reload
```

### Access Services Directly

```bash
# PostgreSQL
docker-compose exec postgres psql -U postgres -d momlaunchpad

# Redis
docker-compose exec redis redis-cli

# Backend shell
docker-compose exec backend sh
```

## Production Deployment

**⚠️ Use GitHub Actions for production, not docker-compose.**

See [.github/CICD.md](.github/CICD.md) for:
- Automated CI/CD pipelines
- Server setup instructions
- Deployment procedures
- Rollback strategies
- Monitoring setup

### Why Not Docker Compose in Production?

Docker Compose is designed for development and testing, not production:
- ❌ No high availability
- ❌ No auto-scaling
- ❌ No rolling updates
- ❌ No health check orchestration
- ❌ Limited resource management

### Production Architecture

```
GitHub Actions CI/CD
  ↓
Build & Push Docker Image
  ↓
Deploy to Server via SSH
  ↓
Single Docker Container
  ↓
Nginx Reverse Proxy (SSL)
  ↓
External PostgreSQL (Managed DB)
  ↓
External Redis (Managed Cache)
```

For full production setup, see [.github/CICD.md](.github/CICD.md).

```bash
For full production setup, see [.github/CICD.md](.github/CICD.md).

---

## Docker Commands Reference (Development)

### Container Management

```bash
# Start services
docker-compose up -d

# Stop services
docker-compose down

# Restart specific service
docker-compose restart backend

# View logs
docker-compose logs -f backend
docker-compose logs --tail=100 postgres

# Remove all containers and volumes
docker-compose down -v
```

### Image Management

```bash
# Build image
docker-compose build backend

# Pull latest images
docker-compose pull

# Remove unused images
docker image prune -a
```

### Database Management

```bash
# Backup database
docker-compose exec postgres pg_dump -U postgres momlaunchpad > backup.sql

# Restore database
docker-compose exec -T postgres psql -U postgres momlaunchpad < backup.sql

# Access PostgreSQL shell
docker-compose exec postgres psql -U postgres -d momlaunchpad

# Run SQL file
docker-compose exec -T postgres psql -U postgres -d momlaunchpad < migrations/001_init_schema.sql
```

### Debugging

```bash
# Check container health
docker-compose ps

# Inspect service
docker inspect momlaunchpad-api

# View resource usage
docker stats momlaunchpad-api

# Execute command in container
docker-compose exec backend go version

# Copy files from container
docker cp momlaunchpad-api:/app/logs/app.log ./local-logs/
```

## Environment Variables

### Required

| Variable | Description | Example |
|----------|-------------|---------|
| `DEEPSEEK_API_KEY` | DeepSeek API key | `sk-...` |
| `JWT_SECRET` | JWT signing secret (min 32 chars) | `your-super-secret-key...` |
| `DATABASE_URL` | PostgreSQL connection string | `postgresql://user:pass@host:5432/db` |

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `REDIS_URL` | `redis://redis:6379` | Redis connection string |
| `GIN_MODE` | `release` | Gin mode (`debug`/`release`) |
| `DEEPSEEK_MODEL` | `deepseek-chat` | AI model name |
| `RATE_LIMIT_ENABLED` | `true` | Enable rate limiting |

## Volume Persistence

### Data Volumes

```bash
# List volumes
docker volume ls | grep momlaunchpad

# Inspect volume
docker volume inspect momlaunchpad-be_postgres_data

# Backup volume
docker run --rm -v momlaunchpad-be_postgres_data:/data -v $(pwd):/backup \
  alpine tar czf /backup/postgres-backup.tar.gz -C /data .

# Restore volume
docker run --rm -v momlaunchpad-be_postgres_data:/data -v $(pwd):/backup \
  alpine tar xzf /backup/postgres-backup.tar.gz -C /data
```

## Health Checks

### Backend Health Endpoint

```bash
# Check if server is running
curl http://localhost:8080/health

# Expected response
{
  "status": "ok",
  "timestamp": "2024-03-15T10:30:00Z"
}
```

### Database Health

```bash
# Check PostgreSQL
docker-compose exec postgres pg_isready -U postgres

# Check connection from backend
docker-compose exec backend nc -zv postgres 5432
```

### Redis Health

```bash
# Ping Redis
docker-compose exec redis redis-cli ping

# Check from backend
docker-compose exec backend nc -zv redis 6379
```

## Scaling

### Horizontal Scaling

```bash
# Scale backend to 3 instances
docker-compose up -d --scale backend=3

# Use with load balancer (Nginx example)
# Add to nginx.conf:
upstream backend {
    server localhost:8080;
    server localhost:8081;
    server localhost:8082;
}
```

### Resource Limits

```yaml
# Add to docker-compose.yml
services:
  backend:
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

## Monitoring

### Container Logs

```bash
# Follow logs with timestamps
docker-compose logs -f -t backend

# Filter logs
docker-compose logs backend | grep "ERROR"

# Export logs
docker-compose logs backend > backend-logs.txt
```

### Metrics

```bash
# Real-time resource usage
docker stats momlaunchpad-api momlaunchpad-db

# Disk usage
docker system df
```

## Security Best Practices

### 1. Use Secrets Management

```bash
# Use Docker secrets (Swarm mode)
echo "my_secret_key" | docker secret create jwt_secret -

# Reference in docker-compose.yml
secrets:
  jwt_secret:
    external: true
```

### 2. Run as Non-Root User

The Dockerfile already creates and uses a non-root user:
```dockerfile
USER appuser
```

### 3. Scan for Vulnerabilities

```bash
# Scan image
docker scan momlaunchpad-api:latest

# Use Trivy
trivy image momlaunchpad-api:latest
```

### 4. Use Multi-Stage Builds

Already implemented in Dockerfile - builds with Go 1.21, runs on Alpine.

## Troubleshooting

### Backend Won't Start

```bash
# Check logs
docker-compose logs backend

# Common issues:
# 1. Missing environment variables
docker-compose exec backend env | grep DEEPSEEK

# 2. Database not ready
docker-compose logs postgres

# 3. Port already in use
lsof -i :8080
```

### Database Connection Failed

```bash
# Test connection
docker-compose exec backend nc -zv postgres 5432

# Check database exists
docker-compose exec postgres psql -U postgres -l

# Verify credentials
docker-compose exec postgres psql -U postgres -d momlaunchpad -c "SELECT 1;"
```

### WebSocket Connection Issues

```bash
# Check if backend is listening
docker-compose exec backend netstat -tuln | grep 8080

# Test WebSocket
websocat ws://localhost:8080/ws/chat?token=YOUR_JWT_TOKEN

# Check CORS headers
curl -H "Origin: http://localhost:3000" -I http://localhost:8080/ws/chat
```

### Out of Memory

```bash
# Check memory usage
docker stats momlaunchpad-api

# Increase memory limit in docker-compose.yml
deploy:
  resources:
    limits:
      memory: 1G
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Build and Push Docker Image

on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Build Docker image
        run: docker build -t momlaunchpad-api:${{ github.sha }} .
      
      - name: Run tests
        run: docker run momlaunchpad-api:${{ github.sha }} go test ./...
      
      - name: Push to registry
        run: |
          echo ${{ secrets.DOCKER_PASSWORD }} | docker login -u ${{ secrets.DOCKER_USERNAME }} --password-stdin
          docker push momlaunchpad-api:${{ github.sha }}
```

## Performance Tuning

### PostgreSQL

```yaml
# Add to docker-compose.yml
postgres:
  environment:
    POSTGRES_INITDB_ARGS: "-c shared_buffers=256MB -c max_connections=200"
```

### Redis

```yaml
redis:
  command: redis-server --maxmemory 256mb --maxmemory-policy allkeys-lru
```

### Backend

```yaml
backend:
  environment:
    GOMAXPROCS: 4
    GOGC: 100
```

## Next Steps

1. Set up monitoring (Prometheus/Grafana)
2. Configure log aggregation (ELK stack)
3. Implement backup automation
4. Set up SSL/TLS with Let's Encrypt
5. Configure CDN for static assets

## Support

For issues, see:
- [Backend Spec](BACKEND_SPEC.md)
- [API Documentation](API.md)
- [WebSocket Guide](WEBSOCKET_GUIDE.md)
