#!/usr/bin/env bash
# Build the anydoc static archive that the Go package links at build time via
# cgo, for a single target.
#
# The archive is a Rust build artifact (~30 MB), built from the pinned anydoc
# crate (see Cargo.toml). scripts/build-all.sh runs this for every supported
# platform and stages the archives into lib/<platform>/; the CI workflow
# (`.github/workflows/build-libs.yml`) does the same and pushes them to main.
#
# Usage:
#   scripts/build-anydoc-lib.sh                # host platform, plain cargo
#   TARGET=aarch64-unknown-linux-gnu scripts/build-anydoc-lib.sh
#   TARGET=... BUILD_TOOL=cross scripts/build-anydoc-lib.sh    # cross (Docker)
#   TARGET=... BUILD_TOOL=zigbuild scripts/build-anydoc-lib.sh # cargo-zigbuild
#
# BUILD_TOOL selects the Rust build driver:
#   cargo    (default) plain cargo; the target must be installed and linkable
#   cross    cross (https://github.com/cross-rs/cross); requires Docker
#   zigbuild cargo-zigbuild (https://github.com/rust-cross/cargo-zigbuild);
#            bundles zig, needs no Docker and no per-target toolchains
set -euo pipefail

# Black box: when set -e kills the script on a failing command, report the
# line and exit code. The Windows runner has failed here silently (exit 101,
# no message) twice; this pinpoints the exact command next time.
trap 'echo "error: build-anydoc-lib.sh failed at line $LINENO (exit code $?)" >&2' ERR

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"

if ! command -v cargo >/dev/null 2>&1; then
  echo "error: cargo not found. Install Rust (https://rustup.rs) to build the anydoc archive." >&2
  exit 1
fi

# The Rust target triple decides the archive; the directory name mirrors what
# the cgo LDFLAGS in the platform-specific Go files expect.
target=${TARGET:-$(rustc -vV | sed -n 's/^host: //p')}
case "$target" in
  x86_64-apple-darwin) lib_dir=darwin_amd64 ;;
  aarch64-apple-darwin) lib_dir=darwin_arm64 ;;
  x86_64-pc-windows-msvc) lib_dir=windows_amd64 ;;
  x86_64-unknown-linux-gnu) lib_dir=linux_amd64 ;;
  aarch64-unknown-linux-gnu) lib_dir=linux_arm64 ;;
  *)
    echo "error: unsupported target '$target'." >&2
    echo "Supported: {x86_64,aarch64}-apple-darwin, {x86_64,aarch64}-unknown-linux-gnu, x86_64-pc-windows-msvc" >&2
    exit 1
    ;;
esac

# The MSVC target produces a COFF archive named anydoc_go.lib; every other
# target produces the GNU archive libanydoc_go.a. Windows builds with MSVC
# (matching the WeKnora binding): the MSVC toolchain cannot be redistributed,
# so that target only builds on a native Windows host, and Go's cgo links the
# .lib with mingw gcc.
case "$target" in
  x86_64-pc-windows-msvc) lib_name=anydoc_go.lib ;;
  *) lib_name=libanydoc_go.a ;;
esac

# The crate version to patch is read from Cargo.toml rather than repeated here,
# so an upgrade is a one-line change to the `anydoc = "=X.Y.Z"` pin.
anydoc_version=$(sed -n 's/^anydoc = "=\([0-9][^"]*\)".*/\1/p' Cargo.toml | head -1)
if [ -z "$anydoc_version" ]; then
  echo "error: no pinned 'anydoc = \"=X.Y.Z\"' dependency in Cargo.toml" >&2
  exit 1
fi

# document_to_markdown is private in published anydoc. Copy the pinned version
# and re-export that one function so anydoc-go can keep the official serializer.
prepare_patched_anydoc() {
  local dest="patched-anydoc"
  # The marker records which version was patched: after a version bump the
  # copy left by the previous build is the wrong crate, and reusing it would
  # fail deep inside cargo's patch resolution instead of here. Cargo.toml is
  # required too: a truncated leftover with only the marker would otherwise
  # skip the copy and fail with "failed to read .../Cargo.toml".
  if [ -f "$dest/.patched" ] && [ "$(cat "$dest/.patched")" = "$anydoc_version" ] && [ -f "$dest/Cargo.toml" ]; then
    return
  fi

  local src=""
  local cargo_home="${CARGO_HOME:-$HOME/.cargo}"
  src=$(find "$cargo_home/registry/src" -maxdepth 2 -type d -name "anydoc-$anydoc_version" 2>/dev/null | head -1 || true)

  if [ -z "$src" ]; then
    local tarball=".anydoc-$anydoc_version.crate"
    echo "Fetching anydoc $anydoc_version to patch document_to_markdown into the public API"
    local rc=0
    # rsproxy.cn (the China mirror) can be slow or unreachable from GitHub's
    # Azure runners; static.crates.io is the canonical fallback. Retry
    # transient failures and report the exit code instead of dying silently.
    echo "Downloading from rsproxy.cn"
    if ! curl --retry 3 --retry-all-errors -fsSL "https://rsproxy.cn/api/v1/crates/anydoc/$anydoc_version/download" -o "$tarball"; then
      rc=$?
      echo "Downloading from static.crates.io (rsproxy.cn failed with $rc)"
      if ! curl --retry 3 --retry-all-errors -fsSL "https://static.crates.io/crates/anydoc/anydoc-$anydoc_version.crate" -o "$tarball"; then
        rc=$?
        echo "error: failed to download anydoc $anydoc_version from rsproxy.cn and static.crates.io (last curl exit code $rc)" >&2
        exit 1
      fi
    fi
    echo "Downloaded $(wc -c < "$tarball") bytes"
    if [ ! -s "$tarball" ] || [ "$(wc -c < "$tarball")" -lt 100000 ]; then
      echo "error: downloaded $tarball is missing or suspiciously small" >&2
      exit 1
    fi
    local unpack=".anydoc-unpack"
    echo "Unpacking $tarball"
    rm -rf "$unpack"
    mkdir -p "$unpack"
    tar -xzf "$tarball" -C "$unpack"
    echo "Unpacked; patching document_to_markdown"
    src=$(find "$unpack" -maxdepth 1 -type d -name "anydoc-$anydoc_version" | head -1)
    if [ -z "$src" ]; then
      echo "error: unpacked anydoc-$anydoc_version crate is missing" >&2
      exit 1
    fi
  fi

  rm -rf "$dest"
  cp -R "$src" "$dest"
  if ! grep -q '^use render::markdown::document_to_markdown;$' "$dest/src/lib.rs"; then
    echo "error: anydoc $anydoc_version lib.rs no longer has the expected document_to_markdown import" >&2
    exit 1
  fi
  sed -i.bak 's/^use render::markdown::document_to_markdown;/pub use render::markdown::document_to_markdown;/' "$dest/src/lib.rs"
  rm -f "$dest/src/lib.rs.bak"
  printf '%s\n' "$anydoc_version" > "$dest/.patched"
  rm -rf ".anydoc-unpack" ".anydoc-$anydoc_version.crate"
}

# version.go is what the Go package reports as its Version on every parsed
# document, so a bump that misses it would mislabel the whole knowledge base.
go_version=$(sed -n 's/^const Version = "\([^"]*\)".*/\1/p' version.go | head -1)
if [ "$go_version" != "$anydoc_version" ]; then
  echo "error: version.go says '$go_version' but Cargo.toml pins anydoc '$anydoc_version'." >&2
  echo "Bump both: the Go constant is the version recorded with parsed documents." >&2
  exit 1
fi

build_tool=${BUILD_TOOL:-cargo}
case "$build_tool" in
  cargo | cross | zigbuild) ;;
  *)
    echo "error: BUILD_TOOL must be cargo, cross, or zigbuild (got '$build_tool')" >&2
    exit 1
    ;;
esac

if [ "$build_tool" = cross ] && ! command -v cross >/dev/null 2>&1; then
  echo "error: BUILD_TOOL=cross but the cross binary is not installed (cargo install cross)." >&2
  exit 1
fi
if [ "$build_tool" = zigbuild ] && ! command -v cargo-zigbuild >/dev/null 2>&1; then
  echo "error: BUILD_TOOL=zigbuild but cargo-zigbuild is not installed (cargo install cargo-zigbuild)." >&2
  exit 1
fi

# Plain cargo needs the rustup target installed for cross-target builds.
if [ "$build_tool" = cargo ] && [ "$target" != "$(rustc -vV | sed -n 's/^host: //p')" ] && command -v rustup >/dev/null 2>&1; then
  echo "Installing rustup target $target"
  rustup target add "$target"
fi

echo "Building anydoc $anydoc_version archive for $target ($build_tool)"
prepare_patched_anydoc

# --locked: build exactly the dependency versions in the committed Cargo.lock.
# The lockfile is committed by the CI workflow after its first run; before
# that, the first build resolves and writes it.
locked=""
if [ -f Cargo.lock ]; then locked="--locked"; fi

# Trim the archive without touching runtime behavior:
# - -C link-dead-code=no: drop code unreachable from the exported ABI (the
#   #[no_mangle] functions are the roots), so unused paths of the dependency
#   tree never enter the archive
# - -C force-unwind-tables=no: skip unwind tables for functions that cannot
#   unwind. Panic = unwind is kept: the ABI's panic containment (guarded)
#   depends on it, and landing pads are still emitted where needed.
export RUSTFLAGS="${RUSTFLAGS:-} -C link-dead-code=no -C force-unwind-tables=no"

# Rebuild our own crate unconditionally: rust-cache restores target/ across
# runs, and a poisoned fingerprint there is exactly the kind of staleness this
# workflow must not ship (a stale archive links fine for every symbol except
# the ones a newer ABI added). Dependencies stay cached, so this costs little.
"$build_tool" clean --release -p anydoc-go >/dev/null 2>&1

"$build_tool" build --release $locked --target "$target"

dest="lib/$lib_dir"
mkdir -p "$dest"
cp "target/$target/release/$lib_name" "$dest/$lib_name"

# The header declares the ABI, so the archive must export every declared
# function. A stale archive would otherwise surface only much later, as a Go
# link failure in the workflow's smoke test; catch it here, next to the
# artifact itself. Skipped where nm is unavailable (the Windows MSVC runner).
if command -v nm >/dev/null 2>&1; then
  # nm can exit non-zero on members it cannot read (for example when the
  # toolchain's nm predates the LLVM that produced the archive); it still
  # prints every symbol it managed to read. `|| true` keeps that partial
  # success from tripping set -e; the symbol comparison below stays
  # authoritative. An empty result fails loudly instead of silently.
  symbols=$(nm -gU "$dest/$lib_name" 2>/dev/null) || true
  exported=$(printf '%s\n' "$symbols" | awk '{print $NF}' | sed 's/^_//' | grep '^anydoc_' | grep -v ':$' | sort -u) || true
  declared=$(grep -o '^[a-z_* ]*anydoc_[a-z_0-9]*(' include/anydoc.h | grep -o 'anydoc_[a-z_0-9]*' | sort -u) || true
  if [ -z "$declared" ]; then
    echo "error: no anydoc_* function declarations found in include/anydoc.h" >&2
    exit 1
  fi
  missing=""
  for sym in $declared; do
    if ! printf '%s\n' "$exported" | grep -qx "$sym"; then
      missing="$missing $sym"
    fi
  done
  if [ -n "$missing" ]; then
    echo "error: $dest/$lib_name is missing declared ABI symbols:$missing" >&2
    echo "The archive is stale or the rebuild was skipped; delete target/ and rebuild." >&2
    exit 1
  fi
fi

# The archive must end exactly at its last member. A duplicated or appended
# tail (seen once already: CI artifact-merge corruption) parses as garbage
# member headers and fails at link time with "archive member invalid control
# bits"; catch it here instead. Skipped where python3 is unavailable.
if command -v python3 >/dev/null 2>&1; then
  python3 - "$dest/$lib_name" <<'PYEOF'
import sys
path = sys.argv[1]
with open(path, "rb") as f:
    data = f.read()
if not data.startswith(b"!<arch>\n"):
    sys.exit(f"error: {path} does not start with an ar magic header")
pos = 8
while pos < len(data):
    if len(data) - pos < 60:
        sys.exit(f"error: {path} has {len(data) - pos} stray bytes after the last member")
    size_field = data[pos + 48:pos + 58].rstrip(b" ")
    try:
        size = int(size_field)
    except ValueError:
        sys.exit(f"error: {path} member header at offset {pos} has a non-numeric size field")
    pos += 60 + size + (size & 1)  # members are padded to even length
if pos != len(data):
    sys.exit(f"error: {path} is {len(data) - pos} bytes longer than its members declare")
PYEOF
fi

echo "Wrote $dest/$lib_name"
