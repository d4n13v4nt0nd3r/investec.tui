#!/usr/bin/env python3
"""Extract the zebra mark from the supplied artwork.

The source is black line art on an opaque white background. The app icon needs
the reverse: an opaque white mark on transparency, so it can be composited onto
the dark tile. This reads the source luminance as coverage, which keeps the
anti-aliased edges intact instead of hard-thresholding them into jaggies.

Reads packaging/source/tui-zebra-logo.png, the original supplied artwork, so
the icons can be rebuilt from scratch without anything outside the repo. Pass a
different path as the first argument to use another source image.

Run from the repo root:

    python3 packaging/tools/extract_mark.py
"""

from __future__ import annotations

import sys
from pathlib import Path

from PIL import Image

REPO = Path(__file__).resolve().parents[2]
SRC = REPO / "packaging" / "source" / "tui-zebra-logo.png"
OUT = REPO / "packaging" / "source" / "zebra-mark.png"

# Luminance at or above this counts as pure background. Anything below scales
# smoothly into the alpha channel so edges stay soft.
WHITE_FLOOR = 250


def main() -> int:
    src = Path(sys.argv[1]) if len(sys.argv) > 1 else SRC
    if not src.is_file():
        print(f"source artwork not found: {src}", file=sys.stderr)
        return 1

    img = Image.open(src).convert("L")

    # Ink coverage: black (0) -> fully opaque, white (255) -> fully clear.
    alpha = img.point(lambda v: 0 if v >= WHITE_FLOOR else 255 - v)

    mark = Image.new("RGBA", img.size, (255, 255, 255, 0))
    mark.putalpha(alpha)

    box = mark.getbbox()
    if box is None:
        print("artwork is blank after thresholding", file=sys.stderr)
        return 1
    mark = mark.crop(box)

    OUT.parent.mkdir(parents=True, exist_ok=True)
    mark.save(OUT)

    w, h = mark.size
    print(f"source     {img.size[0]}x{img.size[1]}")
    print(f"bbox       {box}")
    print(f"mark       {w}x{h}  (aspect {w / h:.4f})")
    print(f"written    {OUT.relative_to(REPO)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
