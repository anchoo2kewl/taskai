Deploy TaskAI to all environments: staging → UAT → production.

## Usage
```
/deploy "feat: your commit message here"
```

If no message is provided as `$ARGUMENTS`, use a generic message based on recent changes.

## Rules
- UAT is ALWAYS promoted from staging (main), never developed on directly
- Production is ALWAYS promoted from main after staging is verified
- Never manually SSH to servers — always use GitHub workflows

## Steps

### 1. Deploy to Staging
Run `./script/server deploy "$ARGUMENTS"` (commits uncommitted changes, pushes to main, triggers GitHub Actions staging deploy).

If there are no uncommitted changes and `$ARGUMENTS` is empty, just push: `git push origin main`.

### 2. Promote to UAT
UAT must be a mirror of main — force-push via GitHub API (UAT branch is protected):

```bash
# 1. Enable force pushes
gh api repos/anchoo2kewl/taskai/branches/uat/protection -X PUT \
  -H "Accept: application/vnd.github+json" \
  --input - <<'EOF'
{"required_status_checks":{"strict":true,"contexts":[]},"enforce_admins":false,"required_pull_request_reviews":null,"restrictions":null,"allow_force_pushes":true,"required_linear_history":false}
EOF

# 2. Force-push main to uat
git push origin main:uat --force

# 3. Re-disable force pushes
gh api repos/anchoo2kewl/taskai/branches/uat/protection -X PUT \
  -H "Accept: application/vnd.github+json" \
  --input - <<'EOF'
{"required_status_checks":{"strict":true,"contexts":[]},"enforce_admins":false,"required_pull_request_reviews":null,"restrictions":null,"allow_force_pushes":false,"required_linear_history":false}
EOF
```

### 3. Promote to Production
```bash
./script/server promote
```

You can chain steps 2 and 3 into a single shell command:
```bash
gh api repos/anchoo2kewl/taskai/branches/uat/protection -X PUT \
  -H "Accept: application/vnd.github+json" \
  --input - <<'EOF'
{"required_status_checks":{"strict":true,"contexts":[]},"enforce_admins":false,"required_pull_request_reviews":null,"restrictions":null,"allow_force_pushes":true,"required_linear_history":false}
EOF
git push origin main:uat --force && \
gh api repos/anchoo2kewl/taskai/branches/uat/protection -X PUT \
  -H "Accept: application/vnd.github+json" \
  --input - <<'EOF'
{"required_status_checks":{"strict":true,"contexts":[]},"enforce_admins":false,"required_pull_request_reviews":null,"restrictions":null,"allow_force_pushes":false,"required_linear_history":false}
EOF
./script/server promote
```

### 4. Report Status
Show a summary table:
| Environment | Action | URL |
|---|---|---|
| Staging | Deployed via push to main | https://staging.taskai.cc |
| UAT | Force-promoted from main | https://uat.taskai.cc |
| Production | GitHub Actions triggered | https://taskai.cc |

Monitor: https://github.com/anchoo2kewl/taskai/actions
