#!/usr/bin/env bash
set -euo pipefail
SRC=~/workspace/Synapse/memory-backend/
DST=~/workspace/Mycelian/mycelian-memory/server/
/opt/homebrew/bin/rsync -aHv --delete --info=progress2 "$SRC" "$DST"
