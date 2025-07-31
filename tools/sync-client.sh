#!/usr/bin/env bash
set -euo pipefail
SRC=~/workspace/Synapse/memory-client/
DST=~/workspace/Mycelian/mycelian-memory/clients/go
/opt/homebrew/bin/rsync -aHv --delete --info=progress2 "$SRC" "$DST"
