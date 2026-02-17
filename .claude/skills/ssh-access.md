# SSH Access to Production Server

## Server Details
- **Host**: `ubuntu@31.97.102.48`
- **Staging Path**: `/home/ubuntu/taskai-staging`
- **Production Path**: `/home/ubuntu/taskai`

## Common SSH Commands

### Check Container Status
```bash
# Staging
ssh ubuntu@31.97.102.48 "cd /home/ubuntu/taskai-staging && docker compose ps"

# Production
ssh ubuntu@31.97.102.48 "cd /home/ubuntu/taskai && docker compose ps"
```

### View Logs
```bash
# API logs
ssh ubuntu@31.97.102.48 "cd /home/ubuntu/taskai && docker compose logs api --tail 50"

# Web logs
ssh ubuntu@31.97.102.48 "cd /home/ubuntu/taskai && docker compose logs web --tail 50"

# Follow logs in real-time
ssh ubuntu@31.97.102.48 "cd /home/ubuntu/taskai && docker compose logs -f api"
```

### Rebuild and Restart Services
```bash
# Rebuild without cache
ssh ubuntu@31.97.102.48 "cd /home/ubuntu/taskai && docker compose build --no-cache"

# Start services
ssh ubuntu@31.97.102.48 "cd /home/ubuntu/taskai && docker compose up -d"

# Restart services
ssh ubuntu@31.97.102.48 "cd /home/ubuntu/taskai && docker compose restart"

# Stop services
ssh ubuntu@31.97.102.48 "cd /home/ubuntu/taskai && docker compose down"
```

### Check Git Status
```bash
ssh ubuntu@31.97.102.48 "cd /home/ubuntu/taskai/source && git log --oneline -5"
ssh ubuntu@31.97.102.48 "cd /home/ubuntu/taskai/source && git status"
```

### Database Operations
```bash
# Access SQLite database
ssh ubuntu@31.97.102.48 "cd /home/ubuntu/taskai && docker compose exec api sqlite3 /data/taskai.db"

# Run a query
ssh ubuntu@31.97.102.48 "cd /home/ubuntu/taskai && docker compose exec api sqlite3 /data/taskai.db 'SELECT * FROM swim_lanes LIMIT 5;'"
```

### Health Checks
```bash
# Check API health
curl -s https://taskai.cc/api/health

# Check staging API health
curl -s https://staging.taskai.cc/api/health

# Check web UI
curl -I https://taskai.cc
```

## Preferred Deployment Workflow

Use `./script/server deploy` instead of manual SSH commands. The script handles:
1. Commit and push to GitHub
2. CI tests run automatically
3. Staging auto-deploys on CI success
4. Use `./script/server promote` for production

### Quick Deploy (when webhook fails)

```bash
# Staging
ssh ubuntu@31.97.102.48 "cd /home/ubuntu/taskai-staging/source && git pull origin main && cd .. && docker compose build --no-cache && docker compose up -d"

# Production
ssh ubuntu@31.97.102.48 "cd /home/ubuntu/taskai/source && git pull origin main && cd .. && docker compose build --no-cache && docker compose up -d"
```

## Troubleshooting

### Containers not running
```bash
ssh ubuntu@31.97.102.48 "cd /home/ubuntu/taskai && docker compose logs --tail 100"
ssh ubuntu@31.97.102.48 "cd /home/ubuntu/taskai && docker compose down && docker compose build --no-cache && docker compose up -d"
```

### Migrations not applied
- Migrations run automatically on API startup
- Check logs: `docker compose logs api | grep -i migration`

### 502 Bad Gateway
- Usually means containers aren't running
- Check with: `docker compose ps`
- Restart with: `docker compose restart`

## Webhook Information

The webhook server runs on the production server and automatically:
1. Pulls latest code from GitHub when CI passes
2. Rebuilds Docker containers
3. Restarts services

**Webhook health**: https://webhook.biswas.me/health
