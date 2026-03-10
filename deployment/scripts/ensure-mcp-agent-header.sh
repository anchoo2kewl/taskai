#!/bin/bash
# Ensure the MCP nginx config forwards X-Agent-Name header to the MCP container.
# Without this, AI agent attribution doesn't work (comments show as the user, not the agent).
#
# Idempotent: skips if already configured.
# Usage: sudo ./ensure-mcp-agent-header.sh <mcp_domain>
# Example: sudo ./ensure-mcp-agent-header.sh mcp.taskai.cc
set -e

DOMAIN="${1:?Usage: ensure-mcp-agent-header.sh <mcp_domain>}"
CONF="/etc/nginx/sites-available/$DOMAIN"

if [ ! -f "$CONF" ]; then
    echo "No nginx config found at $CONF, skipping"
    exit 0
fi

if grep -q 'X-Agent-Name' "$CONF"; then
    echo "X-Agent-Name header already configured for $DOMAIN"
    exit 0
fi

echo "Adding X-Agent-Name header forwarding to $DOMAIN..."

# Insert after the X-API-Key line
sed -i '/proxy_set_header X-API-Key/a\        proxy_set_header X-Agent-Name $http_x_agent_name;' "$CONF"

nginx -t
systemctl reload nginx
echo "X-Agent-Name header added and nginx reloaded for $DOMAIN"
