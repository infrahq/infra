#!/usr/bin/env bash

set -eu -o pipefail

echo "Running Infra connector install on $DESTINATION_NAME"

PACKAGE_PATH=/work/dist/infra_0.0.0_amd64.deb
INFRA_SERVER_URL=test-server-1


# step=install-package
apt-get update && apt-get install --no-install-recommends -y \
    openssh-server procps

dpkg -i "${PACKAGE_PATH}"
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
name: $DESTINATION_NAME
endpointAddr: "$DESTINATION_ADDR"
server:
  url: "$INFRA_SERVER_URL"
  accessKey: "$INFRA_ACCESS_KEY"
  trustedCertificate: /work/internal/server/testdata/pki/ca.crt
EOF


echo "Starting infra service"
systemctl start infra

echo "Starting sshd service"
systemctl start ssh
