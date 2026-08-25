#!/usr/bin/env bash
# Build the static archives for every supported platform and stage them into
# lib/<platform>/.
#
# Platform matrix (see README):
#   linux/amd64    x86_64-unknown-linux-gnu
#   linux/arm64    aarch64-unknown-linux-gnu
#   darwin/amd64   x86_64-apple-darwin
#   darwin/arm64   aarch64-apple-darwin
#   windows/amd64  x86_64-pc-windows-gnu
#
# Tool selection:
#   - macOS targets always build with plain cargo on a macOS host: Apple's SDK
#     cannot be redistributed, so neither cross nor zig can target them from
#     Linux. On non-macOS hosts they are skipped with a warning; the CI
#     workflow builds them on macOS runners.
#   - Linux/Windows targets prefer `cross` (Docker) when available, then
#     `cargo-zigbuild`, then plain cargo (which needs the rustup target and a
#     cross linker installed).
#
# Usage:
#   scripts/build-all.sh                             # every platform
#   scripts/build-all.sh linux_amd64 darwin_arm64    # a subset, by lib dir
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"

if ! command -v cargo >/dev/null 2>&1; then
  echo "error: cargo not found. Install Rust (https://rustup.rs)." >&2
  exit 1
fi

linux_windows_targets=(
  x86_64-unknown-linux-gnu
  aarch64-unknown-linux-gnu
  x86_64-pc-windows-gnu
)
darwin_targets=(
  x86_64-apple-darwin
  aarch64-apple-darwin
)

# Pick the cross toolchain for the Linux/Windows targets.
if command -v cross >/dev/null 2>&1 && command -v docker >/dev/null 2>&1; then
  tool=cross
elif command -v cargo-zigbuild >/dev/null 2>&1; then
  tool=zigbuild
else
  echo "note: neither cross (with Docker) nor cargo-zigbuild found; using plain cargo." >&2
  echo "note: plain cargo needs the rustup target and a cross linker per target." >&2
  tool=cargo
fi
echo "Using '$tool' for Linux/Windows targets."

for target in "${linux_windows_targets[@]}" "${darwin_targets[@]}"; do
  case "$target" in
    x86_64-apple-darwin) dir=darwin_amd64 ;;
    aarch64-apple-darwin) dir=darwin_arm64 ;;
    x86_64-pc-windows-gnu) dir=windows_amd64 ;;
    x86_64-unknown-linux-gnu) dir=linux_amd64 ;;
    aarch64-unknown-linux-gnu) dir=linux_arm64 ;;
  esac

  # Optional positional arguments select a subset, e.g. `linux_amd64`.
  if [ "$#" -gt 0 ]; then
    skip=1
    for want in "$@"; do
      if [ "$want" = "$dir" ]; then skip=0; break; fi
    done
    if [ "$skip" = 1 ]; then continue; fi
  fi

  case "$target" in
    *apple-darwin)
      if [ "$(uname -s)" != "Darwin" ]; then
        echo "skip  $dir ($target): Apple's SDK cannot be redistributed; build on a macOS host or in CI"
        continue
      fi
      BUILD_TOOL=cargo TARGET="$target" "$repo_root/scripts/build-anydoc-lib.sh"
      ;;
    *)
      BUILD_TOOL="$tool" TARGET="$target" "$repo_root/scripts/build-anydoc-lib.sh"
      ;;
  esac
done

echo "Done. Archives under lib/:"
ls -lh lib/*/libanydoc_go.a 2>/dev/null || true
