#!/usr/bin/env bash

set -eu -o pipefail


# step=install-package
dpkg -i "${PACKAGE_PATH}"
apt-get update && apt-get install openssh-server
mkdir -p /run/sshd


# step=write-sshd-config
cat << EOF > /etc/ssh/sshd_config.d/infra.conf
Match group infra-users
    AuthorizedKeysFile none
    PasswordAuthentication no
    AuthorizedKeysCommand /usr/local/sbin/infra sshd auth-keys %u %f
    AuthorizedKeysCommandUser nobody
EOF


# step=write-connector.yaml
cat << EOF > /etc/infra/connector.yaml
kind: ssh
name: testing-ssh
endpointAddr: "$DESTINATION_ADDR"
server:
  url: "$INFRA_SERVER_URL"
  accessKey: "$INFRA_ACCESS_KEY"
EOF

# TODO: exec s6
