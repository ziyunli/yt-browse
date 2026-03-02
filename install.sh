#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"
go build -o yt-browse ./cmd/yt-browse/
mv yt-browse ~/bin/
echo "Installed yt-browse to ~/bin/yt-browse"
