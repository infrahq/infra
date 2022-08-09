#!/bin/sh

if [ "$VERCEL_ENV" = "production" ] ; then
  # Proceed with the build
  exit 1
else
  # only build if website or docs files have changes
  git diff --quiet HEAD^ HEAD ../docs .
fi
