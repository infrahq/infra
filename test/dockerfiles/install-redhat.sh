#!/usr/bin/env bash

set -eu -o pipefail

echo "Running Infra connector install on $DESTINATION_NAME"

PACKAGE_PATH=/work/dist/infra-0.0.0.x86_64.rpm
INFRA_SERVER_URL=test-server-1


# step=install-package
yum install -y openssh-server
yum install -y "${PACKAGE_PATH}"

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
systemctl start sshd
