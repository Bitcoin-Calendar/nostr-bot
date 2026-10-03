#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" != "" && "${1:-}" != "--probe" && "${1:-}" != "--preview" ]]; then
  echo "Usage: check.sh [--probe|--preview [--date YYYY-MM-DD]]" >&2
  exit 2
fi
if [[ $# -eq 0 ]]; then set -- --check; fi
exec systemd-run --uid=bitcal --gid=bitcal --wait --collect --pipe --quiet \
  --property=EnvironmentFile=/etc/bitcal/nostr-bot.env \
  --property=StateDirectory=bitcal-nostr --property=StateDirectoryMode=0700 \
  --property=UMask=0077 --property=NoNewPrivileges=true --property=PrivateTmp=true \
  --property=ProtectSystem=strict --property=ProtectHome=true \
  '--property=RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6' \
  /srv/bitcal/bots/nostr/nostr_bot "$@"
