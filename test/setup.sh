#!/usr/bin/env bash
set -eu -o pipefail

TTY=
if [ -t 1 ]; then TTY=-t; fi

docker exec -i $TTY \
  -e DESTINATION_NAME=ubuntu \
  -e DESTINATION_ADDR=127.0.0.1:8220 \
  -e INFRA_ACCESS_KEY=dest000001.ubuntuubuntuubuntuubuntu \
  test-destination_ubuntu-1 /work/test/dockerfiles/install-debian.sh

docker exec -i $TTY \
  -e DESTINATION_NAME=debian \
  -e DESTINATION_ADDR=127.0.0.1:8221 \
  -e INFRA_ACCESS_KEY=dest000002.debiandebiandebiandebian \
  test-destination_debian-1 /work/test/dockerfiles/install-debian.sh

docker exec -i $TTY \
  -e DESTINATION_NAME=redhat \
  -e DESTINATION_ADDR=127.0.0.1:8222 \
  -e INFRA_ACCESS_KEY=dest000003.redhatredhatredhatredhat \
  test-destination_redhat-1 /work/test/dockerfiles/install-redhat.sh
