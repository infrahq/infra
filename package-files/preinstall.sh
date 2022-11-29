#!/bin/sh

HOMEDIR='/etc/infra'

if command -v dpkg >/dev/null; then
  NOLOGIN=/usr/sbin/nologin
elif command -v rpm >/dev/null; then
  NOLOGIN=/sbin/nologin
fi

set -eu

status() { echo "$*" >&2; }

if ! getent group infra >/dev/null; then
  status 'creating group "infra"'
  groupadd --system infra
fi

if ! getent passwd infra >/dev/null; then
  status 'creating user "infra"'
  useradd --system --shell "$NOLOGIN" --home-dir "$HOMEDIR" --no-create-home --no-user-group --comment 'Infra Agent' infra
  usermod --group infra infra
  usermod --lock infra
fi

if ! getent group infra-users >/dev/null; then
  status 'creating group "infra-users"'
  groupadd infra-users
fi
