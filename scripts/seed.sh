#!/usr/bin/env bash
# Load demo users, posts, and follows for local development.
set -euo pipefail

GATEWAY=${GATEWAY_URL:-http://localhost:8080}

echo "Seeding demo data against $GATEWAY ..."
# TODO: curl -X POST $GATEWAY/api/v1/auth/register ...
echo "seed done"
