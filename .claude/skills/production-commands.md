# Production Management Commands

Quick reference for managing TaskAI staging and production environments.

## Health & Status

```bash
# Full health check (API, Web, MCP, Database)
./script/server staging health
./script/server prod health

# Service status
./script/server staging status
./script/server prod status

# Quick API health
curl https://staging.taskai.cc/api/health
curl https://taskai.cc/api/health
```

## Service Control

```bash
# Restart all services
./script/server staging restart
./script/server prod restart

# Stop services
./script/server staging stop
./script/server prod stop

# Start services
./script/server staging start
./script/server prod start
```

## Logs

```bash
# View API logs (last 50 lines)
./script/server staging logs api
./script/server prod logs api

# View web logs
./script/server staging logs web
./script/server prod logs web

# View all logs
./script/server staging logs
./script/server prod logs
```

## Database Operations

```bash
# Query users
./script/server db-query "SELECT id, email, is_admin, created_at FROM users ORDER BY created_at DESC LIMIT 10;"

# Count users
./script/server db-query "SELECT COUNT(*) as total_users FROM users;"

# View API keys
./script/server db-query "SELECT id, user_id, name, key_prefix, created_at, last_used_at FROM api_keys ORDER BY created_at DESC;"

# Check projects
./script/server db-query "SELECT id, name, owner_id, created_at FROM projects ORDER BY created_at DESC LIMIT 10;"

# View tasks
./script/server db-query "SELECT id, title, status, project_id FROM tasks ORDER BY created_at DESC LIMIT 10;"
```

## Admin Management

```bash
# List all admins
./script/server staging admin list
./script/server prod admin list

# Make user admin
./script/server staging admin create user@example.com
./script/server prod admin create user@example.com

# Revoke admin
./script/server staging admin revoke user@example.com
./script/server prod admin revoke user@example.com
```

## Quick Troubleshooting

### API Not Responding

```bash
# Check logs
./script/server prod logs api

# Check container status
./script/server prod status

# Restart
./script/server prod restart
```

### Database Issues

```bash
# Check tables
./script/server db-query ".tables"

# Check schema
./script/server db-query ".schema users"
```

## Performance Monitoring

```bash
# Check response time
time curl https://taskai.cc/api/health

# Database size
./script/server db-query "SELECT page_count * page_size as size FROM pragma_page_count(), pragma_page_size();"

# User activity
./script/server db-query "SELECT COUNT(*) as active_users FROM users WHERE created_at > datetime('now', '-7 days');"
```

## Production URLs

- **Web UI**: https://taskai.cc
- **API Health**: https://taskai.cc/api/health
- **API Docs**: https://taskai.cc/api/openapi
- **Staging Web**: https://staging.taskai.cc
- **MCP Production**: https://mcp.taskai.cc
- **MCP Staging**: https://mcp.staging.taskai.cc
- **SonarQube**: https://sonar.taskai.cc

## Server Details

- **Server**: ubuntu@31.97.102.48
- **Staging Path**: /home/ubuntu/taskai-staging
- **Production Path**: /home/ubuntu/taskai
- **Staging Domain**: staging.taskai.cc
- **Production Domain**: taskai.cc
