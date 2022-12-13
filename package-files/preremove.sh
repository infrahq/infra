#!/bin/sh

set -eu

systemctl_stop_service() {
  if command -v systemctl >/dev/null; then
    systemctl stop infra || true
    systemctl disable infra || true
  fi
}

dpkg_prerm() {
  case $1 in
    remove) systemctl_stop_service ;;
  esac
}

rpm_prerm() {
  case $1 in
    0) systemctl_stop_service ;;
  esac
}

if command -v dpkg >/dev/null; then
  dpkg_prerm $1
elif command -v rpm >/dev/null; then
  rpm_prerm $1
else
  systemctl_stop_service
fi
