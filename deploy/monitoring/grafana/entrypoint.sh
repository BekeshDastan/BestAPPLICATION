#!/bin/sh
set -e

TPLFILE=/etc/grafana/provisioning/alerting/contactpoints.yml.tpl
OUTFILE=/etc/grafana/provisioning/alerting/contactpoints.yml

sed \
  -e "s|\${TELEGRAM_BOT_TOKEN}|${TELEGRAM_BOT_TOKEN}|g" \
  -e "s|\${TELEGRAM_CHAT_ID}|${TELEGRAM_CHAT_ID}|g" \
  "$TPLFILE" > "$OUTFILE"

exec /run.sh "$@"
