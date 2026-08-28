#!/usr/bin/env python3
"""Check that a built Windows .exe really carries the app icon.

Linking a malformed .syso often succeeds quietly and simply produces an
executable with no usable icon, so this walks the resource directory of a real
PE and compares what it finds against packaging/windows/app.ico.

    GOOS=windows GOARCH=amd64 go build -o /tmp/app.exe .
    python3 packaging/tools/verify_syso.py /tmp/app.exe
"""

from __future__ import annotations

import struct
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
ICO = REPO / "packaging" / "windows" / "app.ico"

RT_ICON = 3
RT_GROUP_ICON = 14


class PE:
    def __init__(self, blob: bytes):
        self.blob = blob
        if blob[:2] != b"MZ":
            raise ValueError("not a PE: missing MZ header")
        pe_off = struct.unpack_from("<I", blob, 0x3C)[0]
        if blob[pe_off : pe_off + 4] != b"PE\0\0":
            raise ValueError("not a PE: missing PE signature")

        coff = pe_off + 4
        (
            self.machine,
            n_sections,
            _ts,
            _sym,
            _nsym,
            opt_size,
            _chars,
        ) = struct.unpack_from("<HHIIIHH", blob, coff)

        opt = coff + 20
        magic = struct.unpack_from("<H", blob, opt)[0]
        # Data directories sit after the optional header's fixed part, which
        # differs in size between PE32 and PE32+.
        dd_off = opt + (112 if magic == 0x20B else 96)
        self.res_rva, self.res_size = struct.unpack_from("<II", blob, dd_off + 8 * 2)

        self.sections = []
        sect = opt + opt_size
        for i in range(n_sections):
            name, vsize, vaddr, rawsize, rawptr = struct.unpack_from(
                "<8sIIII", blob, sect + 40 * i
            )
            self.sections.append(
                (name.rstrip(b"\0").decode(), vaddr, vsize, rawptr, rawsize)
            )

    def rva_to_offset(self, rva: int) -> int:
        for _name, vaddr, vsize, rawptr, rawsize in self.sections:
            if vaddr <= rva < vaddr + max(vsize, rawsize):
                return rawptr + (rva - vaddr)
        raise ValueError(f"RVA {rva:#x} is not inside any section")

    def at(self, rva: int, size: int) -> bytes:
        off = self.rva_to_offset(rva)
        return self.blob[off : off + size]


def walk_dir(pe: PE, base_rva: int, dir_rva: int, depth: int = 0):
    """Yield (path_of_ids, data_rva, size) for every leaf in the tree."""
    off = pe.rva_to_offset(dir_rva)
    n_named, n_id = struct.unpack_from("<HH", pe.blob, off + 12)
    for i in range(n_named + n_id):
        ident, value = struct.unpack_from("<II", pe.blob, off + 16 + 8 * i)
        if value & 0x80000000:
            child = base_rva + (value & 0x7FFFFFFF)
            for path, rva, size in walk_dir(pe, base_rva, child, depth + 1):
                yield (ident,) + path, rva, size
        else:
            entry = pe.rva_to_offset(base_rva + value)
            data_rva, size, _cp, _res = struct.unpack_from("<IIII", pe.blob, entry)
            yield (ident,), data_rva, size


def main() -> int:
    if len(sys.argv) != 2:
        print(__doc__)
        return 2
    exe = Path(sys.argv[1])
    pe = PE(exe.read_bytes())

    if not pe.res_rva:
        print("FAIL: no resource directory in the executable")
        return 1

    leaves = list(walk_dir(pe, pe.res_rva, pe.res_rva))
    icons = {p[1]: (rva, size) for p, rva, size in leaves if p[0] == RT_ICON}
    groups = {p[1]: (rva, size) for p, rva, size in leaves if p[0] == RT_GROUP_ICON}

    print(f"machine        {pe.machine:#06x}")
    print(f"resource dir   rva {pe.res_rva:#x}, {pe.res_size} bytes")
    print(f"RT_ICON        {len(icons)} entries: ids {sorted(icons)}")
    print(f"RT_GROUP_ICON  {len(groups)} entries: ids {sorted(groups)}")

    if not groups:
        print("FAIL: no icon group, Explorer would show the default icon")
        return 1

    # Compare against the source .ico.
    src = ICO.read_bytes()
    _res, _type, count = struct.unpack_from("<HHH", src, 0)
    expected = {}
    for i in range(count):
        size, offset = struct.unpack_from("<II", src, 6 + 16 * i + 8)
        expected[i + 1] = src[offset : offset + size]

    if len(icons) != count:
        print(f"FAIL: {count} images in app.ico but {len(icons)} in the exe")
        return 1

    for ident, (rva, size) in sorted(icons.items()):
        got = pe.at(rva, size)
        want = expected[ident]
        status = "ok" if got == want else "MISMATCH"
        if got != want:
            print(f"FAIL: icon {ident} differs ({len(got)} vs {len(want)} bytes)")
            return 1
        # A correctly relocated DIB starts with a 40-byte BITMAPINFOHEADER,
        # and the 256px entry is a PNG.
        head = "PNG" if got[:4] == b"\x89PNG" else f"DIB {struct.unpack_from('<I', got, 0)[0]}"
        print(f"  icon {ident}: {size:>7} bytes  {head:<8} {status}")

    # The group must reference exactly the ids we found.
    grp_rva, grp_size = next(iter(groups.values()))
    grp = pe.at(grp_rva, grp_size)
    _r, _t, gcount = struct.unpack_from("<HHH", grp, 0)
    ids = [struct.unpack_from("<H", grp, 6 + 14 * i + 12)[0] for i in range(gcount)]
    print(f"  group references ids {ids}")
    if sorted(ids) != sorted(icons):
        print("FAIL: icon group references ids that are not present")
        return 1

    print("PASS: icon resources are present and byte-identical to app.ico")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
