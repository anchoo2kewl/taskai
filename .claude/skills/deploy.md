# Deploy to Production

Deploy TaskAI to production server with automated verification.

## Usage

```bash
./script/server deploy [commit_message]
```

## What it does

1. **Commits changes** - Adds all uncommitted files and creates a commit
2. **Pushes to GitHub** - Pushes the main branch to origin
3. **Waits for CI + webhook** - GitHub Actions CI runs, then staging auto-deploys
4. **Health checks** - Verifies API, Web UI, and MCP are responding
5. **Reports status** - Shows deployment success with URLs or error details

## Examples

```bash
# Deploy with auto-generated message
./script/server deploy

# Deploy with custom message
./script/server deploy "feat: add new feature"

# Deploy bug fix
./script/server deploy "fix: resolve login issue"
```

## Default Commit Format

If no message provided, uses:
```
Deploy: automated deployment

Co-Authored-By: Claude <noreply@anthropic.com>
```

## Verification Process

The script performs comprehensive checks:

- API health endpoint responding
- Web UI accessible
- MCP server responding
- Service health status (up to 20 attempts, 5s intervals)

## Production URLs

After successful deployment to staging:

- **Staging Web**: https://staging.taskai.cc
- **Staging API**: https://staging.taskai.cc/api/health

To promote to production:

- **Production Web**: https://taskai.cc
- **Production API**: https://taskai.cc/api/health

## Troubleshooting

If deployment fails:

```bash
# Check container status
./script/server staging status

# View API logs
./script/server staging logs api

# View web logs
./script/server staging logs web

# Manual restart
./script/server staging restart
```

## Related Commands

```bash
# Check health without deploying
./script/server staging health
./script/server prod health

# Restart services
./script/server staging restart
./script/server prod restart

# View real-time logs
./script/server staging logs api
./script/server prod logs api

# Check service status
./script/server staging status
./script/server prod status

# Query production database
./script/server db-query "SELECT COUNT(*) FROM users;"
```

## Deployment Workflow

1. **Local Development**
   ```bash
   # Make changes
   # Test locally
   ./script/server test
   ```

2. **Deploy to Staging**
   ```bash
   ./script/server deploy "feat: your feature description"
   ```

3. **Verify Staging**
   ```bash
   ./script/server staging health
   ```

4. **Promote to Production**
   ```bash
   ./script/server promote
   ```

5. **Verify Production**
   ```bash
   ./script/server prod health
   ```

## Best Practices

- Test locally before deploying (`./script/server test`)
- Use descriptive commit messages
- Check staging health before promoting to production
- Monitor logs for errors

## Architecture

```
Local Machine → GitHub → CI Tests → Staging (auto)
                                        ↓
                                  Verify staging
                                        ↓
                              Promote → Production
```

## Environment

- **Remote Server**: ubuntu@31.97.102.48
- **Staging Path**: /home/ubuntu/taskai-staging
- **Production Path**: /home/ubuntu/taskai
- **Staging Domain**: staging.taskai.cc
- **Production Domain**: taskai.cc
- **Services**: Docker Compose (api + web + mcp)
- **Deployment**: CI-gated webhook (staging), manual promotion (production)
