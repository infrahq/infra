#!/bin/sh

HOMEDIR='/etc/infra'

set -eu

if [ -f "$HOMEDIR/connector.yaml" ]; then
  if command -v systemctl >/dev/null && [ "$(systemctl is-system-running)" != "offline" ]; then
    systemctl daemon-reload
    systemctl restart infra
  fi
fi
