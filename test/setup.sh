#!/usr/bin/env bash
set -eu -o pipefail

TTY=
if [ -t 1 ]; then TTY=-t; fi

docker exec -i $TTY test-destination_ubuntu-1 /work/test/dockerfiles/install-ubuntu.sh
