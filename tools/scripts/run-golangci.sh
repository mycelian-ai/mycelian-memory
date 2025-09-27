#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GO_WORK_FILE="$ROOT_DIR/../go.work"

if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "golangci-lint not found in PATH" >&2
  exit 1
fi

if [ ! -f "$GO_WORK_FILE" ]; then
  echo "go.work not found at $GO_WORK_FILE" >&2
  exit 1
fi

status=0
modules=()
while IFS= read -r line; do
  mod_path="${line#./}"
  modules+=("$mod_path")
done < <(awk '/^use / { print $2 }' "$GO_WORK_FILE")

for module in "${modules[@]}"; do
  module_dir="$ROOT_DIR/../$module"
  if [ ! -d "$module_dir" ]; then
    echo "skip: module directory $module_dir not found" >&2
    continue
  fi

  # Skip non-Go modules (no go.mod)
  if [ ! -f "$module_dir/go.mod" ]; then
    echo "skip: $module has no go.mod" >&2
    continue
  fi

  # Only lint modules that declare a golangci config to avoid legacy lint debt.
  if [ ! -f "$module_dir/.golangci.yml" ] && [ ! -f "$module_dir/.golangci.yaml" ]; then
    echo "skip: $module has no golangci config" >&2
    continue
  fi

  echo "golangci-lint: running in $module"
  pushd "$module_dir" >/dev/null
  config_args=()
  if [ -f "$module_dir/.golangci.yml" ]; then
    config_args+=("--config" "$module_dir/.golangci.yml")
  elif [ -f "$module_dir/.golangci.yaml" ]; then
    config_args+=("--config" "$module_dir/.golangci.yaml")
  fi

  if [ ${#config_args[@]} -gt 0 ]; then
    if ! golangci-lint run "${config_args[@]}" ./...; then
      status=1
    fi
  else
    if ! golangci-lint run ./...; then
      status=1
    fi
  fi
  popd >/dev/null
  echo
done

exit $status
