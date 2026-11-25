#!/usr/bin/env bash
# Remove LLVM LTO IR sections from a Rust static archive so that cgo can link
# it with any Go release.
#
# Go 1.26.11 and 1.27.4 hardened the cgo linker driver: before passing a
# library to the C linker, the go command parses every archive member and
# rejects any that carries LLVM LTO intermediate representation
# (cmd/go/internal/work/security.go). Linking an unstripped Rust archive fails
# with, for example:
#
#   github.com/nbbug/anydoc-go: disallowed LTO section ".llvmbc" in
#   #cgo LDFLAGS: .../lib/linux_amd64/libanydoc_go.a
#
# Rust's precompiled standard library is built with embedded bitcode, so a Rust
# staticlib carries those sections no matter what this crate's LTO profile says.
# The IR exists only for cross-language LTO, which a cgo link never performs;
# every member also holds the machine code the linker uses, so removing the IR
# changes no exported symbol.
#
# Later Go patch releases re-allow these sections, but the archives are
# downloaded and linked by users on whatever Go version they run, so the
# shipped bytes have to stay linkable on the versions that reject them.
#
# Usage:
#   scripts/strip-llvm-lto.sh <archive>          strip in place
#   scripts/strip-llvm-lto.sh --check <archive>  verify only, change nothing
#
# Every format is rewritten member by member with llvm-style objcopy:
#   ELF/COFF  objcopy --remove-section=.llvmbc ... then re-archive
#   Mach-O    llvm-objcopy --remove-section=__LLVM,__bitcode ...
#
# MSVC archives need one extra step: rustc bundles import objects (and import
# stubs) for the raw-dylib system DLLs, so a static library contains several
# members with the same name (advapi32.dll, bcryptprimitives.dll, ...). ar
# cannot unpack those without losing all but one, so they are unpacked by index
# with python3 and the PE import objects are carried through untouched -- they
# have no sections and objcopy cannot parse them.
#
# Mach-O keeps the bitcode in the __LLVM segment. The Apple linker only drops
# that segment from `ld -r` output when it is a segment of its own; rustc's
# objects keep it inside the object segment, and newer Xcode linkers preserve
# it there, so the object has to be rewritten directly instead of relinked.
#
# Tools: objcopy for the rewrite, objdump to name ELF/COFF sections, ar to
# unpack and repack the archive (python3 instead of ar x for MSVC archives),
# and Apple's otool (in the Xcode command line tools) to name Mach-O sections.
# Prefer objcopy/objdump/ar from the Rust `llvm-tools` component: unlike a host
# binutils they read every Rust target and format, so one Linux host can strip
# an arm64 archive. TOOL_PREFIX selects a cross binutils instead, e.g.
# TOOL_PREFIX=aarch64-linux-gnu-.
set -euo pipefail

prog=${0##*/}
script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
die() {
  printf 'error: %s: %s\n' "$prog" "$*" >&2
  exit 1
}

mode=strip
case "${1:-}" in
  --check) mode=check; shift ;;
  --strip) shift ;;
esac
[ "$#" -eq 1 ] || die "usage: $prog [--check] <path/to/static-archive>"
[ -f "$1" ] || die "no such archive: $1"
archive=$(cd "$(dirname "$1")" && pwd)/$(basename "$1")

# ---------------------------------------------------------------------------
# Tool discovery
# ---------------------------------------------------------------------------

sysroot=""
if command -v rustc >/dev/null 2>&1; then
  sysroot=$(rustc --print sysroot 2>/dev/null | tr -d '\r' || true)
  # rustc prints a Win32 path (C:\\...) on Windows; Git Bash's test -x wants
  # the POSIX form.
  if [ -n "$sysroot" ] && command -v cygpath >/dev/null 2>&1; then
    sysroot=$(cygpath -u "$sysroot" 2>/dev/null || printf '%s' "$sysroot")
  fi
fi

# rustup_tool NAME: a binary installed by the llvm-tools component, if any.
# These read objects for every target and format, unlike a host binutils. The
# host directory is globbed rather than rebuilt from `rustc -vV`, so a stray
# CR or an unexpected triple cannot hide the tools.
rustup_tool() {
  local name=$1 cand
  [ -n "$sysroot" ] || return 1
  for cand in "$sysroot"/lib/rustlib/*/bin/"$name" "$sysroot"/lib/rustlib/*/bin/"$name.exe"; do
    if [ -x "$cand" ]; then
      printf '%s\n' "$cand"
      return 0
    fi
  done
  return 1
}

# find_tool NAME: llvm-tools first, then PATH.
find_tool() {
  rustup_tool "$1" && return 0
  command -v "$1" 2>/dev/null && return 0
  return 1
}

ar=""
for candidate in llvm-ar "${TOOL_PREFIX:-}ar" ar; do
  ar=$(find_tool "$candidate" || true)
  [ -n "$ar" ] && break
done
[ -n "$ar" ] || die "no ar/llvm-ar found; install binutils or run 'rustup component add llvm-tools'"

work=$(mktemp -d "${TMPDIR:-/tmp}/anydoc-lto.XXXXXX" 2>/dev/null ||
  mktemp -d 2>/dev/null || true)
[ -n "$work" ] && [ -d "$work" ] || die "cannot create a temporary directory"
trap 'rm -rf "$work"' EXIT

# ---------------------------------------------------------------------------
# Format detection: sniff the first member, not the host OS
# ---------------------------------------------------------------------------

# Tool output is CR-stripped throughout: Windows archivers and llvm-objdump
# emit CRLF, and a stray CR would turn member names and section names into
# lookups that silently match nothing.
"$ar" t "$archive" 2>"$work/members.err" | tr -d '\r' >"$work/members.all" ||
  die "cannot list the members of $archive: $(cat "$work/members.err")"
[ -s "$work/members.all" ] ||
  die "$ar listed no members of $archive; it may be the wrong archiver for this format (install llvm-tools, or set TOOL_PREFIX)"

# Archive symbol tables are not objects: the linker members of GNU and MSVC
# archives ('/', '//', and the MSVC second linker member), and the BSD/Mach-O
# '__.SYMDEF*'. Everything else has to be an object we understand.
grep -vE '^(/.*|__\.SYMDEF.*)$' "$work/members.all" >"$work/members" || true
[ -s "$work/members" ] || die "no object members in $archive"

# hex_bytes FILE N / u16_le FILE: read the leading bytes without assuming a
# full coreutils install (Git Bash on Windows does not always ship od).
hex_bytes() {
  if command -v od >/dev/null 2>&1; then
    od -An -tx1 -N"$2" "$1" | tr -d ' \n'
  elif command -v python3 >/dev/null 2>&1; then
    python3 -c 'import sys; sys.stdout.write(open(sys.argv[1], "rb").read(int(sys.argv[2])).hex())' "$1" "$2"
  else
    die "need od or python3 to read the first bytes of $1"
  fi
}

u16_le() {
  if command -v od >/dev/null 2>&1; then
    od -An -tu2 -N2 "$1" | tr -d ' '
  elif command -v python3 >/dev/null 2>&1; then
    python3 -c 'import sys; sys.stdout.write(str(int.from_bytes(open(sys.argv[1], "rb").read(2), "little")))' "$1"
  else
    die "need od or python3 to read the first bytes of $1"
  fi
}

# The first member is not a reliable probe: an MSVC archive interleaves
# regular objects, import stubs, and PE import objects, and only the first two
# kinds start with an object header. Probe members until one is recognisable.
format=""
probes=0
while IFS= read -r probe_member; do
  probes=$((probes + 1))
  [ "$probes" -le 25 ] || break
  "$ar" p "$archive" "$probe_member" >"$work/probe" 2>/dev/null || continue
  [ -s "$work/probe" ] || continue
  case "$(hex_bytes "$work/probe" 4)" in
    7f454c46) format=elf; break ;;
    cffaedfe | cefaedfe | feedface | feedfacf | cafebabe | cafebabf) format=macho; break ;;
  esac
  # COFF has no magic; the first field of the file header is the machine type.
  # 332=0x14c i386, 34404=0x8664 x86_64, 43620=0xaa64 arm64, 448=0x1c0 arm,
  # 452=0x1c4 armnt, 512=0x200 ia64, 20530/20580=riscv.
  case "$(u16_le "$work/probe")" in
    332 | 34404 | 43620 | 448 | 452 | 512 | 20530 | 20580 | 25188) format=coff; break ;;
  esac
done <"$work/members"
[ -n "$format" ] || die "unrecognised object format in $archive"

# MSVC archives repeat member names on purpose (see extract_members). Anywhere
# else a repeat would make ar x drop a member, so refuse such an archive.
if [ "$format" != coff ]; then
  duplicates=$(sort "$work/members" | uniq -d)
  if [ -n "$duplicates" ]; then
    die "member names are not unique ($(printf '%s\n' "$duplicates" | awk 'NR==1') repeats); refusing to repack $archive"
  fi
fi

# ---------------------------------------------------------------------------
# ELF/COFF: objcopy per member
# ---------------------------------------------------------------------------

objcopy=""
objdump=""

# require_objcopy: the section remover, shared by every format.
require_objcopy() {
  [ -n "$objcopy" ] && return 0
  local candidate
  for candidate in llvm-objcopy "${TOOL_PREFIX:-}objcopy" objcopy; do
    objcopy=$(find_tool "$candidate" || true)
    [ -n "$objcopy" ] && return 0
  done
  return 1
}

# require_objdump: only ELF/COFF need it, to name the sections to remove.
require_objdump() {
  [ -n "$objdump" ] && return 0
  local candidate
  for candidate in llvm-objdump "${TOOL_PREFIX:-}objdump" objdump; do
    objdump=$(find_tool "$candidate" || true)
    [ -n "$objdump" ] && return 0
  done
  return 1
}

require_binutils() { require_objcopy && require_objdump; }

# The names the go command rejects (cmd/go/internal/work/security.go). Match
# them exactly: .llvm_addrsig and friends are ordinary LLVM metadata sections,
# not LTO IR, and removing them would be pointless.
# shellcheck disable=SC2016
lto_name_re='^(\.llvmbc|\.llvmcmd|\.llvm\.lto|\.llvm\.linker\.options|\.llvm\.dependent-libraries|\.gnu\.lto_|\.gnu\.debuglto_)'

# lto_sections FILE: section names the Go linker rejects for this object.
lto_sections() {
  "$objdump" -h -w "$1" 2>/dev/null | tr -d '\r' | awk -v re="$lto_name_re" '
    { for (i = 1; i <= NF; i++)
        if ($i ~ re) print $i }
  ' | sort -u || true
}

# ---------------------------------------------------------------------------
# Mach-O: llvm-objcopy per member
# ---------------------------------------------------------------------------

macho_otool=""
require_otool() {
  [ -n "$macho_otool" ] && return 0
  if command -v xcrun >/dev/null 2>&1; then
    macho_otool=$(xcrun --find otool 2>/dev/null || true)
  fi
  [ -n "$macho_otool" ] || macho_otool=$(command -v otool 2>/dev/null || true)
  [ -n "$macho_otool" ]
}

# macho_lto_ir FILE: the __LLVM/__GNU_LTO segment lines, if any. A Mach-O
# archive member carries bitcode in the __LLVM segment (section __bitcode),
# which is exactly what the go command rejects.
macho_lto_ir() {
  "$macho_otool" -l "$1" 2>/dev/null | tr -d '\r' |
    grep -E 'segname[[:space:]]+__(LLVM|GNU_LTO)|sectname[[:space:]]+(\.llvmbc|\.llvmcmd|\.llvm\.lto|\.gnu\.lto_|\.gnu\.debuglto_)' || true
}

# macho_lto_sections FILE: the same sections as "segment,section" pairs, the
# form llvm-objcopy --remove-section takes. otool prints each section's own
# segment/section pair under its "Section" header.
macho_lto_sections() {
  "$macho_otool" -l "$1" 2>/dev/null | tr -d '\r' | awk '
    function emit() {
      if (!insection) return
      if (seg ~ /^__(LLVM|GNU_LTO)$/ ||
          sec ~ /^\.llvmbc$/ || sec ~ /^\.llvmcmd$/ || sec ~ /^\.llvm\.lto/ ||
          sec ~ /^\.gnu\.lto_/ || sec ~ /^\.gnu\.debuglto_/)
        print seg "," sec
      insection = 0
    }
    /^Section$/ { emit(); insection = 1; sec = ""; seg = ""; next }
    insection && $1 == "sectname" { sec = $2; next }
    insection && $1 == "segname" { seg = $2; next }
    END { emit() }
  '
}

# ---------------------------------------------------------------------------
# Scan: report LTO IR in every member (used by --check and post-strip)
# ---------------------------------------------------------------------------

# extract_members ARCHIVE DIR: unpack every member into DIR and print one
# "<path>\t<kind>" line per member, in archive order. kind is "object" when the
# member is a regular object objcopy can rewrite, "import" for a PE import
# object, and "other" otherwise, which the caller refuses rather than ship.
#
# ELF and Mach-O archives unpack with ar: their member names are unique, and
# the extraction is checked against the archive listing so nothing is lost.
#
# MSVC archives cannot: rustc bundles import objects and import stubs for the
# raw-dylib system DLLs, so one name can appear several times and ar x would
# silently keep only the last copy, dropping symbols the linker needs. Those
# archives are unpacked by index (msvc-lib-members.py) under unique file names.
# Member names are cosmetic for linking; the repack regenerates the archive
# symbol table that links symbols to members.
extract_members() {
  local target=$1 dir=$2 member missing="" f
  rm -rf "$dir" && mkdir -p "$dir"
  if [ "$format" = coff ]; then
    python3 "$script_dir/msvc-lib-members.py" "$target" "$dir" ||
      die "cannot unpack $target: not a readable static library"
    return 0
  fi
  (cd "$dir" && "$ar" x "$target") || die "cannot extract $target with $ar"
  while IFS= read -r member; do
    [ -f "$dir/$member" ] || missing="$missing $member"
  done <"$work/members"
  [ -z "$missing" ] || die "ar x did not extract from $target:$missing"
  for f in "$dir"/*; do
    [ -f "$f" ] || continue
    case "${f##*/}" in *SYMDEF*) rm -f "$f"; continue ;; esac
    printf '%s\tobject\n' "$f"
  done
}

scan_archive() {
  local target=$1 dir="$work/scan" member kind base found=0 hits section sections
  extract_members "$target" "$dir" >"$work/scan.manifest" ||
    die "cannot unpack $target"
  while IFS=$'\t' read -r member kind; do
    [ "$kind" = object ] || continue
    base=${member##*/}
    case "$format" in
      macho)
        hits=$(macho_lto_ir "$member")
        if [ -n "$hits" ]; then
          printf '%s: %s\n' "$base" "$(printf '%s' "$hits" | awk 'NR==1')"
          found=1
        fi
        ;;
      *)
        sections=$(lto_sections "$member")
        while IFS= read -r section; do
          [ -n "$section" ] || continue
          printf '%s: %s\n' "$base" "$section"
          found=1
        done <<<"$sections"
        ;;
    esac
  done <"$work/scan.manifest"
  return $((found == 0 ? 0 : 1))
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

case "$format" in
  elf | coff)
    if [ "$mode" = strip ]; then
      require_binutils ||
        die "no objcopy/objdump found; install binutils or run 'rustup component add llvm-tools'"
    else
      require_objdump ||
        die "no objdump found; install binutils or run 'rustup component add llvm-tools'"
    fi
    if [ "$format" = coff ]; then
      command -v python3 >/dev/null 2>&1 ||
        die "no python3 found; MSVC archives are unpacked by index because their member names repeat"
    fi
    ;;
  macho)
    require_otool ||
      die "no otool found; Mach-O archives can only be inspected on macOS (Xcode command line tools)"
    if [ "$mode" = strip ]; then
      require_objcopy ||
        die "no llvm-objcopy found; run 'rustup component add llvm-tools'"
    fi
    ;;
esac

if [ "$mode" = check ]; then
  if findings=$(scan_archive "$archive"); then
    printf '%s: %s carries no LLVM LTO IR\n' "$prog" "$archive"
    exit 0
  fi
  printf '%s: %s carries LLVM LTO IR that the go command rejects:\n' "$prog" "$archive" >&2
  printf '%s\n' "$findings" >&2
  exit 1
fi

mkdir -p "$work/out"
extract_members "$archive" "$work/in" >"$work/manifest" ||
  die "cannot unpack $archive"
[ -s "$work/manifest" ] || die "no members unpacked from $archive"

changed=0

while IFS=$'\t' read -r member kind; do
  base=${member##*/}
  case "$kind" in
    import)
      # PE import object for a raw-dylib system DLL: no sections, nothing for
      # objcopy to remove, and objcopy cannot parse it. Carry it over as is.
      cp "$member" "$work/out/$base"
      continue
      ;;
    object) ;;
    *) die "member '$base' of $archive is neither a COFF object nor a PE import object" ;;
  esac

  case "$format" in
    macho)
      if ! "$macho_otool" -h "$member" >/dev/null 2>&1; then
        die "member '$base' of $archive is not a Mach-O object"
      fi
      if [ -z "$(macho_lto_ir "$member")" ]; then
        cp "$member" "$work/out/$base"
        continue
      fi
      # Rewrite the member without its bitcode sections. llvm-objcopy drops
      # the __LLVM segment once its last section is gone; the machine code and
      # every symbol stay as they are.
      args=()
      pairs=$(macho_lto_sections "$member")
      while IFS= read -r pair; do
        [ -n "$pair" ] && args+=("--remove-section=$pair")
      done <<<"$pairs"
      [ "${#args[@]}" -gt 0 ] ||
        die "member '$base' of $archive carries LTO IR but no section to remove it by"
      "$objcopy" "${args[@]}" "$member" "$work/out/$base" ||
        die "llvm-objcopy failed on member '$base' of $archive"
      [ -z "$(macho_lto_ir "$work/out/$base")" ] ||
        die "the __LLVM segment survived llvm-objcopy on member '$base' of $archive"
      ;;
    *)
      if ! "$objdump" -h "$member" >/dev/null 2>&1; then
        die "member '$base' of $archive is not an object file"
      fi
      sections=$(lto_sections "$member")
      if [ -z "$sections" ]; then
        cp "$member" "$work/out/$base"
        continue
      fi
      args=()
      while IFS= read -r section; do
        [ -n "$section" ] && args+=("--remove-section=$section")
      done <<<"$sections"
      "$objcopy" "${args[@]}" "$member" "$work/out/$base" ||
        die "objcopy failed on member '$base' of $archive"
      remaining=$(lto_sections "$work/out/$base")
      [ -z "$remaining" ] ||
        die "LTO section(s) survived objcopy on member '$base' of $archive: $(printf '%s' "$remaining" | tr '\n' ' ')"
      ;;
  esac
  changed=1
done <"$work/manifest"

if [ "$changed" -eq 0 ]; then
  printf '%s: %s carries no LLVM LTO IR, left untouched\n' "$prog" "$archive"
  exit 0
fi

# Repack in archive order. `ar` regenerates the symbol index, including the
# entries for PE import objects, so the result resolves like the archive Rust
# wrote; what it writes is still a normal COFF static library, which mingw's ld
# (what cgo uses) and LLD both read.
members=()
while IFS=$'\t' read -r member _kind; do
  members+=("${member##*/}")
done <"$work/manifest"
[ "${#members[@]}" -gt 0 ] || die "nothing to repack for $archive"
(cd "$work/out" && "$ar" rcs "$work/stripped" "${members[@]}") ||
  die "cannot repack $archive"

# Gate on the bytes Go will actually see, not on the transform's opinion of
# itself: the repacked archive is rescanned, and only then replaces the input.
if findings=$(scan_archive "$work/stripped"); then :; else
  printf '%s: %s still carries LLVM LTO IR after stripping:\n' "$prog" "$archive" >&2
  printf '%s\n' "$findings" >&2
  exit 1
fi

mv -f "$work/stripped" "$archive"
printf '%s: stripped LLVM LTO IR from %s\n' "$prog" "$archive"
