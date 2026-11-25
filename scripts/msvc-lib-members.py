#!/usr/bin/env python3
"""Unpack an MSVC (or GNU) static library member by member, by index.

`strip-llvm-lto.sh` needs every member of an archive as a separate file so it
can rewrite the ones that carry LLVM LTO IR and repack them. It normally uses
`ar x`, but MSVC static libraries cannot be unpacked that way: rustc bundles
one PE import object per raw-dylib DLL, so a static library legitimately
contains several members with the same name (for example advapi32.dll), and
`ar x` silently keeps only the last one -- dropping import objects whose
symbols the linker then cannot resolve.

This script walks the archive by index instead and writes every member to its
own file, in archive order, under a unique name. Member names are cosmetic for
linking: symbols are resolved through the archive symbol table, which the
repack regenerates.

Usage:
    msvc-lib-members.py <archive> <destdir>

Stdout: one "<path>\t<kind>" line per member, where kind is
    object  a regular COFF object (objcopy can rewrite it)
    import  a PE import object (no sections; carry it through untouched)
    other   anything else (the caller refuses rather than ship it blind)
"""

import os
import struct
import sys

MACHINES = {0x14C, 0x8664, 0xAA64, 0x01C0, 0x01C4, 0x0200, 0x5032, 0x5064}


def fail(message):
    sys.stderr.write("error: %s\n" % message)
    raise SystemExit(1)


def classify(body):
    """Say what a member is, using the PE/COFF header layouts."""
    if len(body) >= 20:
        # PE import object: Sig1=0, Sig2=0xFFFF, Version=0, then the machine.
        sig1, sig2, version, machine = struct.unpack_from("<HHHH", body, 0)
        if sig1 == 0 and sig2 == 0xFFFF and version == 0 and machine in MACHINES:
            size_of_data = struct.unpack_from("<I", body, 12)[0]
            if 20 + size_of_data == len(body):
                return "import"
        # Regular COFF object: machine, section count, optional header size.
        machine, sections = struct.unpack_from("<HH", body, 0)
        optional = struct.unpack_from("<H", body, 16)[0]
        if (
            machine in MACHINES
            and sections > 0
            and 20 + optional + 40 * sections <= len(body)
        ):
            return "object"
    return "other"


def members(data, archive):
    """Yield (name, body) for every real member, in archive order."""
    long_names = b""
    pos = 8
    while pos + 60 <= len(data):
        header = data[pos : pos + 60]
        if header[58:60] != b"`\n":
            fail("malformed archive header at offset %d in %s" % (pos, archive))
        raw = header[0:16].decode("latin1").strip()
        try:
            size = int(header[48:58].decode("latin1").strip())
        except ValueError:
            fail("non-numeric member size at offset %d in %s" % (pos, archive))
        if size < 0 or pos + 60 + size > len(data):
            fail("truncated member at offset %d in %s" % (pos, archive))
        body = data[pos + 60 : pos + 60 + size]
        pos += 60 + size + (size & 1)
        if raw == "/":
            continue  # symbol table; MSVC libraries carry two of them
        if raw == "//":
            long_names = body  # GNU/MSVC long-name table
            continue
        if raw.startswith("__.SYMDEF"):
            continue
        name = raw
        if raw.startswith("/") and raw[1:].isdigit():
            offset = int(raw[1:])
            if offset >= len(long_names):
                fail("long-name offset %d past the table in %s" % (offset, archive))
            name = long_names[offset:].split(b"\0", 1)[0].decode("latin1").rstrip("\n")
        elif raw.startswith("#1/"):
            length = int(raw[3:])
            if length > len(body):
                fail("malformed BSD member name in %s" % archive)
            name = body[:length].decode("latin1").rstrip("\0")
            body = body[length:]
        elif raw.endswith("/"):
            name = raw[:-1]
        yield name, body


def main(argv):
    if len(argv) != 3:
        fail("usage: %s <archive> <destdir>" % os.path.basename(argv[0]))
    archive, dest = argv[1], argv[2]
    try:
        with open(archive, "rb") as handle:
            data = handle.read()
    except OSError as exc:
        fail("cannot read %s: %s" % (archive, exc))
    if not data.startswith(b"!<arch>\n"):
        fail("%s is not an ar archive" % archive)

    unpacked = 0
    for index, (name, body) in enumerate(members(data, archive)):
        safe = "".join(c if c.isalnum() or c in "._-" else "_" for c in name)
        path = os.path.join(dest, "m%04d-%s" % (index, safe or "member"))
        with open(path, "wb") as handle:
            handle.write(body)
        sys.stdout.write("%s\t%s\n" % (path, classify(body)))
        unpacked += 1
    if unpacked == 0:
        fail("no members in %s" % archive)


if __name__ == "__main__":
    main(sys.argv)
