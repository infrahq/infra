#!/usr/bin/env bash
set -eu -o pipefail
export NEXT_BUILD_ID=embed
npm run export --prefix ../../ui
rm -rf ./ui
mkdir -p ./ui
mv ../../ui/out/* ./ui