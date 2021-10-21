#!/bin/sh

gofmt -s -w $1

EXPRESSIONS='s/Api/API/g'

case $(uname -s) in
  Linux) sed -i -e ${EXPRESSIONS// / -e } $1 ;;
  Darwin) sed -i '' -e ${EXPRESSIONS// / -e } $1 ;;
esac
