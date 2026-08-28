#!/usr/bin/env python3
"""Generate the Windows resource objects that give the .exe its icon.

The Go toolchain links any *.syso file in the main package's directory into the
binary. A .syso is just a COFF object file, so embedding an icon means writing
one containing a .rsrc section with an RT_GROUP_ICON and its RT_ICON images.
Doing it here avoids taking a build-time dependency on a Go resource tool.

Writes, at the repo root where the linker looks:

    rsrc_windows_amd64.syso
    rsrc_windows_arm64.syso

Run from the repo root, after build_icons.py:

    python3 packaging/tools/mksyso.py

Layout of the emitted object
----------------------------
    COFF header
    one section header for .rsrc
    section data:  directory tree -> data entries -> icon payloads
    relocations:   one per data entry
    symbol table:  the .rsrc section symbol
    string table:  empty

Each IMAGE_RESOURCE_DATA_ENTRY holds an RVA that is only known once the linker
places the section, so every one of them carries an ADDR32NB relocation against
the section symbol.
"""

from __future__ import annotations

import struct
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
ICO = REPO / "packaging" / "windows" / "app.ico"

RT_ICON = 3
RT_GROUP_ICON = 14
LANG_EN_US = 0x0409
GROUP_ID = 1

# COFF machine types and their "32-bit RVA, no base" relocation.
TARGETS = {
    "amd64": (0x8664, 0x0003),  # IMAGE_REL_AMD64_ADDR32NB
    "arm64": (0xAA64, 0x0002),  # IMAGE_REL_ARM64_ADDR32NB
}

IMAGE_SCN_CNT_INITIALIZED_DATA = 0x00000040
IMAGE_SCN_MEM_READ = 0x40000000
IMAGE_SYM_CLASS_STATIC = 3


def read_ico(path: Path) -> list[tuple[dict, bytes]]:
    """Split an .ico into its directory entries and image payloads."""
    blob = path.read_bytes()
    reserved, kind, count = struct.unpack_from("<HHH", blob, 0)
    if reserved != 0 or kind != 1:
        raise ValueError(f"{path} is not an icon file")

    images = []
    for i in range(count):
        (
            width,
            height,
            colors,
            _res,
            planes,
            bits,
            size,
            offset,
        ) = struct.unpack_from("<BBBBHHII", blob, 6 + 16 * i)
        images.append(
            (
                {
                    "width": width,
                    "height": height,
                    "colors": colors,
                    "planes": planes,
                    "bits": bits,
                },
                blob[offset : offset + size],
            )
        )
    return images


def group_icon_dir(images: list[tuple[dict, bytes]]) -> bytes:
    """Build the RT_GROUP_ICON payload.

    Same shape as the .ico directory except each entry ends with the RT_ICON
    resource id rather than a file offset, making the entry 14 bytes not 16.
    """
    out = bytearray(struct.pack("<HHH", 0, 1, len(images)))
    for i, (meta, data) in enumerate(images, start=1):
        out += struct.pack(
            "<BBBBHHIH",
            meta["width"],
            meta["height"],
            meta["colors"],
            0,
            meta["planes"],
            meta["bits"],
            len(data),
            i,
        )
    return bytes(out)


class Node:
    """One level of the resource directory tree."""

    def __init__(self, entries: list[tuple[int, "Node | int"]]):
        # entries must be sorted by id; Windows binary-searches them.
        self.entries = sorted(entries, key=lambda e: e[0])
        self.offset = 0

    def size(self) -> int:
        return 16 + 8 * len(self.entries)


def build_tree(icon_count: int) -> Node:
    """RT_ICON/<n>/<lang> plus RT_GROUP_ICON/1/<lang>, leaves numbered in order."""
    icons = Node(
        [
            (i, Node([(LANG_EN_US, i - 1)]))
            for i in range(1, icon_count + 1)
        ]
    )
    group = Node([(GROUP_ID, Node([(LANG_EN_US, icon_count)]))])
    return Node([(RT_ICON, icons), (RT_GROUP_ICON, group)])


def walk(node: Node) -> list[Node]:
    """Directories in breadth-first order, which is how they get laid out."""
    order: list[Node] = []
    queue = [node]
    while queue:
        current = queue.pop(0)
        order.append(current)
        for _id, child in current.entries:
            if isinstance(child, Node):
                queue.append(child)
    return order


def align(n: int, to: int = 8) -> int:
    return (n + to - 1) // to * to


def build_rsrc(payloads: list[bytes], reloc_type: int) -> tuple[bytes, bytes]:
    """Return (section data, relocation table) for the resource section."""
    root = build_tree(len(payloads) - 1)  # last payload is the group
    dirs = walk(root)

    # Directories first, in breadth-first order.
    cursor = 0
    for node in dirs:
        node.offset = cursor
        cursor += node.size()

    # Then one IMAGE_RESOURCE_DATA_ENTRY per leaf, in leaf-index order.
    data_entry_base = cursor
    cursor += 16 * len(payloads)

    # Then the payloads themselves.
    payload_offsets = []
    for blob in payloads:
        cursor = align(cursor)
        payload_offsets.append(cursor)
        cursor += len(blob)
    total = cursor

    out = bytearray(total)

    # Directory headers and entries.
    for node in dirs:
        struct.pack_into(
            "<IIHHHH", out, node.offset, 0, 0, 0, 0, 0, len(node.entries)
        )
        pos = node.offset + 16
        for ident, child in node.entries:
            if isinstance(child, Node):
                value = child.offset | 0x80000000
            else:
                value = data_entry_base + 16 * child
            struct.pack_into("<II", out, pos, ident, value)
            pos += 8

    # Data entries. OffsetToData is section-relative here; the relocation turns
    # it into an RVA at link time.
    relocs = bytearray()
    for i, blob in enumerate(payloads):
        entry = data_entry_base + 16 * i
        struct.pack_into("<IIII", out, entry, payload_offsets[i], len(blob), 0, 0)
        out[payload_offsets[i] : payload_offsets[i] + len(blob)] = blob
        relocs += struct.pack("<IIH", entry, 0, reloc_type)

    return bytes(out), bytes(relocs)


def build_syso(payloads: list[bytes], machine: int, reloc_type: int) -> bytes:
    section, relocs = build_rsrc(payloads, reloc_type)
    n_relocs = len(relocs) // 10

    header_size = 20 + 40
    section_ptr = header_size
    reloc_ptr = section_ptr + len(section)
    symbol_ptr = reloc_ptr + len(relocs)

    coff = struct.pack(
        "<HHIIIHH",
        machine,
        1,           # one section
        0,           # timestamp, left zero for reproducible output
        symbol_ptr,
        1,           # one symbol
        0,           # no optional header
        0,           # characteristics
    )

    sect = struct.pack(
        "<8sIIIIIIHHI",
        b".rsrc",
        0,           # VirtualSize
        0,           # VirtualAddress
        len(section),
        section_ptr,
        reloc_ptr if n_relocs else 0,
        0,           # line numbers
        n_relocs,
        0,
        IMAGE_SCN_CNT_INITIALIZED_DATA | IMAGE_SCN_MEM_READ,
    )

    # The single symbol the relocations resolve against: the section itself.
    symbol = struct.pack(
        "<8sIhHBB", b".rsrc", 0, 1, 0, IMAGE_SYM_CLASS_STATIC, 0
    )
    string_table = struct.pack("<I", 4)  # size only; no long names

    return coff + sect + section + relocs + symbol + string_table


def main() -> int:
    if not ICO.is_file():
        raise SystemExit(
            f"missing {ICO.relative_to(REPO)} -- run "
            "packaging/tools/build_icons.py first"
        )

    images = read_ico(ICO)
    payloads = [data for _meta, data in images] + [group_icon_dir(images)]

    for arch, (machine, reloc_type) in TARGETS.items():
        blob = build_syso(payloads, machine, reloc_type)
        out = REPO / f"rsrc_windows_{arch}.syso"
        out.write_bytes(blob)
        print(f"wrote {out.name} ({len(blob)} bytes, {len(images)} icon sizes)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
