#!/bin/bash
# Add zero-downtime deployment support to nginx config:
# - Custom error page for 502/503/504 (friendly "Updating..." page)
# - proxy_next_upstream directives for retry on upstream failure
#
# Idempotent: skips if already configured.
# Usage: sudo ./ensure-zero-downtime.sh <domain>
# Example: sudo ./ensure-zero-downtime.sh staging.taskai.cc
set -e

DOMAIN="${1:?Usage: ensure-zero-downtime.sh <domain>}"
CONF="/etc/nginx/sites-available/$DOMAIN"

if [ ! -f "$CONF" ]; then
    echo "No nginx config found at $CONF, skipping"
    exit 0
fi

CHANGED=0

# 1. Add error_page and /50x.html location if not present
if grep -q 'error_page 502 503 504 /50x.html' "$CONF"; then
    echo "error_page already configured for $DOMAIN"
else
    echo "Adding error_page and /50x.html to $DOMAIN..."
    python3 -c "
import sys

with open('${CONF}') as f:
    content = f.read()

# Build the error page block
error_block = '''
    # Friendly maintenance page during deployments
    error_page 502 503 504 /50x.html;
    location = /50x.html {
        internal;
        default_type text/html;
        return 503 '<!DOCTYPE html><html><head><title>Updating...</title><meta http-equiv=\"refresh\" content=\"5\"></head><body style=\"font-family:system-ui;display:flex;justify-content:center;align-items:center;height:100vh;margin:0;background:#f8fafc\"><div style=\"text-align:center\"><h1 style=\"color:#1e293b\">Updating TaskAI</h1><p style=\"color:#64748b\">Please wait a moment, this page will refresh automatically.</p></div></body></html>';
    }
'''

# Insert before the first location block (after security headers/logging)
marker = '    # Auth endpoints'
if marker not in content:
    marker = '    # API endpoints'
if marker not in content:
    marker = '    location'

idx = content.find(marker)
if idx > 0:
    content = content[:idx] + error_block.strip() + '\n\n    ' + content[idx:]
else:
    print('Could not find insertion point for error_page', file=sys.stderr)
    sys.exit(1)

with open('${CONF}', 'w') as f:
    f.write(content)
"
    CHANGED=1
fi

# 2. Add proxy_next_upstream to location blocks that proxy_pass but lack it
if grep -q 'proxy_next_upstream' "$CONF"; then
    echo "proxy_next_upstream already configured for $DOMAIN"
else
    echo "Adding proxy_next_upstream to proxy locations in $DOMAIN..."
    python3 -c "
import re

with open('${CONF}') as f:
    content = f.read()

# Directive to add after the last proxy_ line in each location block
next_upstream = '''
        proxy_next_upstream error timeout http_502 http_503;
        proxy_next_upstream_timeout 5s;
        proxy_next_upstream_tries 2;'''

# Find all location blocks that have proxy_pass but not proxy_next_upstream
# Strategy: find each proxy_read_timeout or last proxy_ directive and append after it
lines = content.split('\n')
result = []
in_location = False
has_proxy_pass = False
last_proxy_idx = -1
inserted_indices = set()

# First pass: identify insertion points
for i, line in enumerate(lines):
    stripped = line.strip()
    if stripped.startswith('location') and '{' in stripped:
        in_location = True
        has_proxy_pass = False
        last_proxy_idx = -1
    elif in_location and stripped.startswith('proxy_'):
        has_proxy_pass = True
        last_proxy_idx = i
    elif in_location and stripped == '}':
        if has_proxy_pass and last_proxy_idx >= 0:
            inserted_indices.add(last_proxy_idx)
        in_location = False

# Second pass: build result with insertions
for i, line in enumerate(lines):
    result.append(line)
    if i in inserted_indices:
        result.append('')
        for directive_line in next_upstream.strip().split('\n'):
            result.append(directive_line)

with open('${CONF}', 'w') as f:
    f.write('\n'.join(result))
"
    CHANGED=1
fi

if [ "$CHANGED" -eq 1 ]; then
    nginx -t
    systemctl reload nginx
    echo "Zero-downtime directives added and nginx reloaded for $DOMAIN"
else
    echo "No changes needed for $DOMAIN"
fi
