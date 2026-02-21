#!/usr/bin/env bash
# TaskAI adapter for unified project runner
#
# Admin: is_admin = 1 in SQLite (users table)
#   User must exist first (sign up via web UI), then grant admin via sqlite3
#   create-admin signs up via local API then sets is_admin = 1

PROJECT_NAME="taskai"
PROJECT_DOMAIN="taskai.cc"
PROJECT_REPO="anchoo2kewl/taskai"
PROJECT_STACK="Go + React + SQLite"
PROJECT_PORT_BACKEND=8083
PROJECT_PORT_FRONTEND=5174
PROJECT_DB="sqlite"

_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
_run() { (cd "$_dir" && ./script/server "$@"); }

# ── Remote config ────────────────────────────────────────────────
_deploy_path() {
    case "$1" in
        staging) echo "/home/ubuntu/taskai-staging" ;;
        uat)     echo "/home/ubuntu/taskai-uat" ;;
        prod)    echo "/home/ubuntu/taskai" ;;
    esac
}

_api_container() {
    case "$1" in
        staging) echo "taskai-staging-api-1" ;;
        uat)     echo "taskai-uat-api-1" ;;
        prod)    echo "taskai-api-1" ;;
    esac
}

_api_port() {
    case "$1" in
        staging) echo 8083 ;;
        uat)     echo 38888 ;;
        prod)    echo 8082 ;;
    esac
}

# ── Local ─────────────────────────────────────────────────────────
local_start()         { _run local dev; }
local_stop()          { kill_port "$PROJECT_PORT_BACKEND"; kill_port "$PROJECT_PORT_FRONTEND"; }
local_dev()           { _run local dev; }
local_restart()       { local_stop; sleep 1; local_start; }
local_status()        { _run local:status; }
local_logs()          { _run local:logs "$@"; }
local_test()          { _run test; }
local_test_backend()  { _run test-api; }
local_test_frontend() { _run test-web; }
local_db_migrate()    { print_warning "$PROJECT_NAME: SQLite — no migrations"; }
local_db_reset()      { _run local:clean; }
local_db_seed()       { print_warning "$PROJECT_NAME: no seed command"; }
local_users()         { _run local users:list; }
local_create_admin()  { _run local admin create "$1"; }

# ── Docker ────────────────────────────────────────────────────────
docker_start()   { _run local:start; }
docker_stop()    { _run local:stop; }
docker_status()  { _run local:status; }
docker_logs()    { _run local:logs "$@"; }
docker_restart() { docker_stop; docker_start; }

# ── Remote ────────────────────────────────────────────────────────
remote_status() {
    local env="$1" server; server=$(resolve_server "$env") || return 1
    ssh_cmd "$server" "cd '$(_deploy_path "$env")' && docker compose ps"
}

remote_logs() {
    local env="$1"; shift; local server; server=$(resolve_server "$env") || return 1
    ssh_cmd "$server" "cd '$(_deploy_path "$env")' && docker compose logs --tail=100 -f ${1:-}"
}

remote_health() {
    local env="$1" domain="$PROJECT_DOMAIN"
    [[ "$env" == "staging" ]] && domain="staging.$PROJECT_DOMAIN"
    [[ "$env" == "uat" ]]     && domain="uat.$PROJECT_DOMAIN"
    if curl -sf "https://$domain/health" >/dev/null 2>&1; then
        echo -e "  ${GREEN}[healthy]${NC} $PROJECT_NAME  https://$domain"
    else
        echo -e "  ${RED}[down]${NC}    $PROJECT_NAME  https://$domain"
    fi
}

remote_restart() {
    local env="$1" server; server=$(resolve_server "$env") || return 1
    ssh_cmd "$server" "cd '$(_deploy_path "$env")' && docker compose restart"
}

remote_users() {
    local env="$1" server container
    server=$(resolve_server "$env") || return 1
    container=$(_api_container "$env")
    ssh_cmd "$server" "docker exec -u root $container sh -c '
        command -v sqlite3 >/dev/null 2>&1 || apk add --no-cache sqlite >/dev/null 2>&1
        sqlite3 -header -column /data/taskai.db \"SELECT id, email, is_admin, created_at FROM users ORDER BY id;\"
    '"
}

# create-admin <env> <email> <password>
# Signs up via local API (bypasses Cloudflare), then sets is_admin = 1 via sqlite3
remote_create_admin() {
    local env="$1" email="$2" password="$3"
    local server port container

    server=$(resolve_server "$env") || return 1
    port=$(_api_port "$env")
    container=$(_api_container "$env")

    if [[ -z "$email" || -z "$password" ]]; then
        print_error "Usage: <env> create-admin taskai <email> <password>"
        return 1
    fi

    # Step 1: Sign up via local API on the server (bypasses Cloudflare Access)
    print_status "Creating user $email on $env..."
    local signup_result
    signup_result=$(ssh_cmd "$server" "curl -sS -X POST 'http://127.0.0.1:$port/api/auth/signup' \
        -H 'Content-Type: application/json' \
        -d '{\"email\":\"$email\",\"password\":\"$password\"}'") || {
        print_error "Failed to reach API on $env"
        return 1
    }

    if echo "$signup_result" | grep -q "already"; then
        print_warning "User $email already exists — granting admin"
    elif echo "$signup_result" | grep -q '"id"'; then
        print_success "Created user $email"
    else
        print_error "Signup failed: $signup_result"
        return 1
    fi

    # Step 2: Grant admin via sqlite3 inside the api container
    print_status "Granting admin to $email..."
    ssh_cmd "$server" "docker exec -u root $container sh -c '
        command -v sqlite3 >/dev/null 2>&1 || apk add --no-cache sqlite >/dev/null 2>&1
        sqlite3 /data/taskai.db \"UPDATE users SET is_admin = 1 WHERE email = '\"'\"'$email'\"'\"';\"
        sqlite3 -header -column /data/taskai.db \"SELECT id, email, is_admin FROM users WHERE email = '\"'\"'$email'\"'\"';\"
    '" || {
        print_error "Failed to grant admin"
        return 1
    }

    print_success "$email is now admin on $env"
}

# copy-db <from_env> <to_env>
# Copies SQLite file between environments via local temp file
# Uses Docker volume mountpoints on the host — no container stop needed
copy_db() {
    local from_env="$1" to_env="$2"
    local from_server to_server
    local tmp_file="/tmp/taskai-${from_env}-$(date +%s).db"

    from_server=$(resolve_server "$from_env") || return 1
    to_server=$(resolve_server "$to_env") || return 1

    # Find the volume mountpoints on each host
    local from_vol to_vol
    from_vol=$(ssh_cmd "$from_server" "docker volume inspect \$(docker inspect $(_api_container "$from_env") --format '{{range .Mounts}}{{if eq .Destination \"/data\"}}{{.Name}}{{end}}{{end}}') --format '{{.Mountpoint}}'") || {
        print_error "Failed to find source DB volume on $from_env"
        return 1
    }

    to_vol=$(ssh_cmd "$to_server" "docker volume inspect \$(docker inspect $(_api_container "$to_env") --format '{{range .Mounts}}{{if eq .Destination \"/data\"}}{{.Name}}{{end}}{{end}}') --format '{{.Mountpoint}}'") || {
        print_error "Failed to find destination DB volume on $to_env"
        return 1
    }

    # Step 1: Copy source DB to /tmp on source host, then download
    print_status "Downloading DB from $from_env..."
    ssh_cmd "$from_server" "sudo cp '$from_vol/taskai.db' /tmp/taskai-copy.db && sudo chmod 644 /tmp/taskai-copy.db" || {
        print_error "Failed to read DB on $from_env"
        return 1
    }
    scp -o ConnectTimeout=10 "$from_server:/tmp/taskai-copy.db" "$tmp_file" || {
        print_error "Failed to download DB from $from_env"
        return 1
    }
    ssh_cmd "$from_server" "rm -f /tmp/taskai-copy.db" 2>/dev/null
    local db_size
    db_size=$(ls -lh "$tmp_file" | awk '{print $5}')
    print_status "Downloaded: $db_size"

    # Step 2: Upload to /tmp on destination, then copy into volume
    print_status "Uploading DB to $to_env..."
    scp -o ConnectTimeout=10 "$tmp_file" "$to_server:/tmp/taskai-copy.db" || {
        print_error "Failed to upload DB to $to_env"
        return 1
    }
    ssh_cmd "$to_server" "sudo cp /tmp/taskai-copy.db '$to_vol/taskai.db' && rm -f /tmp/taskai-copy.db" || {
        print_error "Failed to write DB on $to_env"
        return 1
    }

    # Step 3: Restart destination so the app picks up the new DB
    print_status "Restarting $to_env..."
    ssh_cmd "$to_server" "cd '$(_deploy_path "$to_env")' && docker compose restart api"

    # Cleanup
    rm -f "$tmp_file"

    print_success "Database copied: $from_env -> $to_env ($db_size)"
}
