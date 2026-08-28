#!/usr/bin/env python3
"""Build the macOS and Windows app icons from the extracted zebra mark.

Renders a white zebra on the app's own background colour, then emits:

    packaging/macos/AppIcon.iconset/   the PNG ladder iconutil turns into .icns
    packaging/windows/app.ico          multi-resolution Windows icon
    packaging/source/preview.png       contact sheet for eyeballing small sizes

Run from the repo root:

    python3 packaging/tools/build_icons.py

Geometry notes
--------------
macOS follows Apple's Big Sur grid: on a 1024 canvas the rounded rectangle is
824x824 with a 185pt corner radius, which is what makes an icon sit correctly
next to its neighbours in the Dock. Windows has no such convention and its
icons are full bleed, so the tile fills the canvas with a gentler radius.

Both platforms grow the mark at small sizes. Holding the large-size proportions
at 16px leaves the zebra an unreadable smudge, and those sizes appear in lists
and menus rather than the Dock, so consistency with the grid matters less than
staying legible.

Small sizes also swap in a simplified mark. The artwork has roughly a dozen
stripes, which cannot be represented in a 16px-wide shape no matter how the
resampling is done -- they average out to flat grey. Below the crossover the
icon therefore uses a solid head silhouette, which stays sharp and still reads
as the same animal.
"""

from __future__ import annotations

import struct
import subprocess
import sys
from io import BytesIO
from pathlib import Path

from PIL import Image, ImageDraw, ImageFilter

REPO = Path(__file__).resolve().parents[2]
MARK = REPO / "packaging" / "source" / "zebra-mark.png"
MARK_SMALL = REPO / "packaging" / "source" / "zebra-mark-small.png"
ICONSET = REPO / "packaging" / "macos" / "AppIcon.iconset"
ICNS = REPO / "packaging" / "macos" / "AppIcon.icns"
ICO = REPO / "packaging" / "windows" / "app.ico"
PREVIEW = REPO / "packaging" / "source" / "preview.png"

BG = (13, 17, 23, 255)          # #0D1117, the app's painted background
FG = (255, 255, 255, 255)       # white mark

# Supersampling factor for the tile mask and mark placement. Corners and the
# mark's diagonals are both resampled down from this, which is what keeps the
# 16px renders from going crunchy.
SS = 4

# The mark's bbox is not its optical centre: the muzzle carries more weight low
# and left, so the shape reads as sitting high unless nudged down slightly.
# Expressed as a fraction of the tile.
NUDGE_Y = 0.015
NUDGE_X = 0.0

# At or below this size the simplified silhouette is used instead of the full
# striped mark. 32px still resolves the stripes; 24px does not.
SIMPLIFY_AT = 24

# Morphological closing radius, in pixels of the ~1000px master, used to merge
# the stripes into one solid head. Tuned to swallow the stripe gaps while
# leaving the ear notch and eye intact.
CLOSE_K = 65


class Profile:
    """Tile and mark proportions for one platform at one size band."""

    def __init__(self, tile_frac: float, radius_frac: float, mark_frac: float):
        self.tile_frac = tile_frac      # tile size / canvas size
        self.radius_frac = radius_frac  # corner radius / tile size
        self.mark_frac = mark_frac      # mark's long edge / tile size


# Apple's grid is 824/1024 = 0.8047 with a 185/824 = 0.2245 radius.
MACOS_LARGE = Profile(tile_frac=0.8047, radius_frac=0.2245, mark_frac=0.66)
MACOS_MID = Profile(tile_frac=0.86, radius_frac=0.2245, mark_frac=0.72)
MACOS_SMALL = Profile(tile_frac=0.94, radius_frac=0.21, mark_frac=0.80)

WINDOWS_LARGE = Profile(tile_frac=1.0, radius_frac=0.10, mark_frac=0.72)
WINDOWS_SMALL = Profile(tile_frac=1.0, radius_frac=0.08, mark_frac=0.82)


def macos_profile(size: int) -> Profile:
    if size <= 32:
        return MACOS_SMALL
    if size <= 64:
        return MACOS_MID
    return MACOS_LARGE


def windows_profile(size: int) -> Profile:
    return WINDOWS_SMALL if size <= 32 else WINDOWS_LARGE


def _morph(alpha: Image.Image, kind: str, k: int) -> Image.Image:
    """Dilate (max) or erode (min) by roughly k pixels.

    Pillow's rank filters take small kernels, so a large radius is reached by
    applying a smaller kernel repeatedly.
    """
    step = 9
    f = ImageFilter.MaxFilter if kind == "max" else ImageFilter.MinFilter
    rounds, rem = divmod(k, step)
    for _ in range(rounds):
        alpha = alpha.filter(f(step))
    if rem > 1:
        alpha = alpha.filter(f(rem if rem % 2 else rem + 1))
    return alpha


def simplify(alpha: Image.Image) -> Image.Image:
    """Collapse the striped mark into a solid head silhouette.

    A closing (dilate then erode) merges the stripes without inflating the
    outline, so the silhouette keeps the original proportions.
    """
    closed = _morph(_morph(alpha, "max", CLOSE_K), "min", CLOSE_K)
    # Harden the edge: the icon is rendered from a supersampled master, so the
    # anti-aliasing is better recreated on the way down than carried through.
    return closed.point(lambda v: 255 if v >= 128 else 0)


def white(alpha: Image.Image) -> Image.Image:
    """A white RGBA image carrying the given alpha."""
    img = Image.new("RGBA", alpha.size, FG)
    img.putalpha(alpha)
    return img.crop(img.getbbox())


def load_marks() -> tuple[Image.Image, Image.Image]:
    """Return the (full, simplified) marks as white-on-transparent images."""
    if not MARK.is_file():
        sys.exit(
            f"missing {MARK.relative_to(REPO)} -- run "
            "packaging/tools/extract_mark.py first"
        )
    alpha = Image.open(MARK).convert("RGBA").getchannel("A")
    full = white(alpha)
    small = white(simplify(alpha))
    small.save(MARK_SMALL)
    return full, small


def render(marks: tuple[Image.Image, Image.Image], size: int, p: Profile) -> Image.Image:
    """Render one icon at `size` px square."""
    full, small = marks
    mark = small if size <= SIMPLIFY_AT else full

    big = size * SS
    canvas = Image.new("RGBA", (big, big), (0, 0, 0, 0))

    tile = round(big * p.tile_frac)
    radius = round(tile * p.radius_frac)
    x0 = (big - tile) // 2
    y0 = (big - tile) // 2

    draw = ImageDraw.Draw(canvas)
    draw.rounded_rectangle(
        [x0, y0, x0 + tile - 1, y0 + tile - 1], radius=radius, fill=BG
    )

    # Scale the mark so its longest edge hits the target fraction of the tile.
    target = tile * p.mark_frac
    scale = target / max(mark.size)
    mw = max(1, round(mark.size[0] * scale))
    mh = max(1, round(mark.size[1] * scale))
    scaled = mark.resize((mw, mh), Image.LANCZOS)

    mx = x0 + (tile - mw) // 2 + round(tile * NUDGE_X)
    my = y0 + (tile - mh) // 2 + round(tile * NUDGE_Y)
    canvas.alpha_composite(scaled, (mx, my))

    return canvas.resize((size, size), Image.LANCZOS)


def build_iconset(marks: tuple[Image.Image, Image.Image]) -> None:
    """Write the .iconset directory iconutil expects."""
    # (filename size, actual pixel size)
    entries = [
        ("icon_16x16.png", 16),
        ("icon_16x16@2x.png", 32),
        ("icon_32x32.png", 32),
        ("icon_32x32@2x.png", 64),
        ("icon_128x128.png", 128),
        ("icon_128x128@2x.png", 256),
        ("icon_256x256.png", 256),
        ("icon_256x256@2x.png", 512),
        ("icon_512x512.png", 512),
        ("icon_512x512@2x.png", 1024),
    ]
    if ICONSET.exists():
        for old in ICONSET.iterdir():
            old.unlink()
    ICONSET.mkdir(parents=True, exist_ok=True)

    cache: dict[int, Image.Image] = {}
    for name, size in entries:
        if size not in cache:
            cache[size] = render(marks, size, macos_profile(size))
        cache[size].save(ICONSET / name)
    print(f"wrote {len(entries)} PNGs to {ICONSET.relative_to(REPO)}")


def make_icns() -> None:
    ICNS.parent.mkdir(parents=True, exist_ok=True)
    subprocess.run(
        ["iconutil", "-c", "icns", str(ICONSET), "-o", str(ICNS)], check=True
    )
    print(f"wrote {ICNS.relative_to(REPO)} ({ICNS.stat().st_size} bytes)")


def dib_bytes(img: Image.Image) -> bytes:
    """Encode an image as the BMP/DIB payload an .ico entry expects.

    Not the same as a .bmp file: there is no file header, the height is
    doubled to cover the AND mask, and rows run bottom-up.
    """
    w, h = img.size
    px = img.load()

    xor = bytearray()
    for y in range(h - 1, -1, -1):
        for x in range(w):
            r, g, b, a = px[x, y]
            xor += bytes((b, g, r, a))

    # 1bpp AND mask, rows padded to 4-byte boundaries. With a real alpha
    # channel present Windows ignores it, so leave it fully opaque (zeroed).
    row_bytes = ((w + 31) // 32) * 4
    and_mask = bytes(row_bytes * h)

    header = struct.pack(
        "<IiiHHIIiiII",
        40,          # biSize
        w,           # biWidth
        h * 2,       # biHeight, XOR + AND
        1,           # biPlanes
        32,          # biBitCount
        0,           # biCompression, BI_RGB
        len(xor) + len(and_mask),
        0, 0, 0, 0,  # resolution and palette counts
    )
    return header + bytes(xor) + and_mask


def write_ico(images: dict[int, Image.Image], path: Path) -> None:
    """Write a multi-resolution .ico.

    256px goes in PNG-compressed to keep the file small, which Vista and later
    understand. Everything below stays as a DIB, which every Windows shell
    surface handles without argument.
    """
    sizes = sorted(images)
    payloads: list[bytes] = []
    for size in sizes:
        img = images[size]
        if size >= 256:
            buf = BytesIO()
            img.save(buf, format="PNG")
            payloads.append(buf.getvalue())
        else:
            payloads.append(dib_bytes(img))

    out = bytearray(struct.pack("<HHH", 0, 1, len(sizes)))  # ICONDIR
    offset = 6 + 16 * len(sizes)
    for size, data in zip(sizes, payloads):
        out += struct.pack(
            "<BBBBHHII",
            0 if size >= 256 else size,  # 0 means 256
            0 if size >= 256 else size,
            0,                            # palette colours
            0,                            # reserved
            1,                            # colour planes
            32,                           # bits per pixel
            len(data),
            offset,
        )
        offset += len(data)
    for data in payloads:
        out += data

    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(out)
    print(f"wrote {path.relative_to(REPO)} ({len(out)} bytes, sizes {sizes})")


def build_preview(marks: tuple[Image.Image, Image.Image]) -> None:
    """Contact sheet: each platform at the sizes that actually get used."""
    sizes = [16, 24, 32, 48, 64, 256]
    pad = 24
    swatch = 256
    width = pad + len(sizes) * (swatch + pad)
    height = pad + 2 * (swatch + pad)

    # Mid grey so both a light and dark surround are approximated.
    sheet = Image.new("RGBA", (width, height), (128, 128, 128, 255))
    for row, profile_for in enumerate((macos_profile, windows_profile)):
        for col, size in enumerate(sizes):
            icon = render(marks, size, profile_for(size))
            # Nearest-neighbour blow-up so pixel-level mush is visible.
            shown = icon.resize((swatch, swatch), Image.NEAREST)
            x = pad + col * (swatch + pad)
            y = pad + row * (swatch + pad)
            sheet.alpha_composite(shown, (x, y))
    sheet.save(PREVIEW)
    print(f"wrote {PREVIEW.relative_to(REPO)} (top row macOS, bottom Windows)")


def main() -> int:
    marks = load_marks()

    build_iconset(marks)
    make_icns()

    ico_sizes = [16, 24, 32, 48, 64, 128, 256]
    write_ico(
        {s: render(marks, s, windows_profile(s)) for s in ico_sizes}, ICO
    )

    build_preview(marks)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
