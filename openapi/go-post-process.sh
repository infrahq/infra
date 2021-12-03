#!/bin/sh

gofmt -s -w $1

case $(uname -s) in
    Linux) sed -i -f $(dirname $0)/substitutions $1 ;;
    Darwin) sed -i '' -f $(dirname $0)/substitutions $1 ;;
esac
