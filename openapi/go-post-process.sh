#!/bin/sh

gofmt -s -w $1

commands() {
    cat <<EOF
s/Api/API/g
s/Id\b/ID/g
s/IdOK/IDOK/gi
s/Ca\b/CA/g
s/CaOK/CAOK/gi
s/\(Get[A-Za-z]\+\)Ok\b/\1OK/g
EOF
}

case $(uname -s) in
  Linux) commands | sed -i -f- $1 ;;
  Darwin) commands | sed -i '' -f- $1 ;;
esac
