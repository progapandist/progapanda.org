#!/usr/bin/env python3
"""Generate the favicon and link-preview image into src/public/.

Run from the repo root: python3 tools/icons.py

Pixel art beats a design tool here: the whole site is a terminal, and every
asset comes out of a grid the same way the welcome banner does. Stdlib only —
the PNG encoder below is shorter than adding a dependency.
"""
import struct
import zlib

INK = (0x0B, 0x0B, 0x0F, 255)
PINK = (0xFF, 0x87, 0xD7, 255)
SNOW = (0xF2, 0xF2, 0xF5, 255)
PAL = {".": PINK, "K": INK, "W": SNOW}

PANDA = [
    "................",
    ".KK..........KK.",
    "KKKK........KKKK",
    "KKKKWWWWWWWWKKKK",
    ".WWWWWWWWWWWWWW.",
    "WWWWWWWWWWWWWWWW",
    "WWKKKWWWWWWKKKWW",
    "WWKWKWWWWWWKWKWW",
    "WWKKKWWWWWWKKKWW",
    "WWWWWWWKKWWWWWWW",
    "WWWWWWWKKWWWWWWW",
    "WWWWWWWWWWWWWWWW",
    ".WWWWWWWWWWWWWW.",
    "..WWWWWWWWWWWW..",
    "....WWWWWWWW....",
    "................",
]

# A face a pixel off-centre is obvious at any size.
assert all(len(r) == len(PANDA) for r in PANDA), "grid must be square"
assert all(r == r[::-1] for r in PANDA), "grid must be symmetric"


def write_png(path, width, height, rows):
    raw = b"".join(b"\x00" + bytes(r) for r in rows)

    def chunk(tag, data):
        return (
            struct.pack(">I", len(data))
            + tag
            + data
            + struct.pack(">I", zlib.crc32(tag + data) & 0xFFFFFFFF)
        )

    blob = (
        b"\x89PNG\r\n\x1a\n"
        + chunk(b"IHDR", struct.pack(">IIBBBBB", width, height, 8, 6, 0, 0, 0))
        + chunk(b"IDAT", zlib.compress(raw, 9))
        + chunk(b"IEND", b"")
    )
    with open(path, "wb") as f:
        f.write(blob)
    return len(blob)


def icon(scale, pad):
    size = len(PANDA) * scale + pad * 2
    rows = []
    for y in range(size):
        row = bytearray()
        gy = (y - pad) // scale
        for x in range(size):
            gx = (x - pad) // scale
            inside = 0 <= gy < len(PANDA) and 0 <= gx < len(PANDA[0])
            row += bytes(PAL[PANDA[gy][gx]] if inside else PINK)
        rows.append(row)
    return size, rows


def svg():
    out = [
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"'
        ' shape-rendering="crispEdges">',
        '<rect width="16" height="16" fill="#ff87d7"/>',
    ]
    for y, line in enumerate(PANDA):
        x = 0
        while x < len(line):
            run = 1
            while x + run < len(line) and line[x + run] == line[x]:
                run += 1
            if line[x] in "KW":
                fill = "#0b0b0f" if line[x] == "K" else "#f2f2f5"
                out.append(
                    f'<rect x="{x}" y="{y}" width="{run}" height="1" fill="{fill}"/>'
                )
            x += run
    return "\n".join(out) + "\n</svg>\n"


def wordmark():
    """The banner art, read straight from the welcome screen it also draws."""
    art, inside = [], False
    with open("visitor/entrypoint.sh", encoding="utf-8") as f:
        for line in f:
            if line.startswith("cat << 'EOF'"):
                inside = True
            elif inside and line.startswith("EOF"):
                break
            elif inside:
                art.append(line.rstrip("\n"))
    width = max(len(l) for l in art)
    return [l.ljust(width) for l in art], width


def preview(path):
    """1200x630 for link previews: the panda over the wordmark, on terminal ink."""
    W, H = 1200, 630
    canvas = [bytearray(bytes(INK) * W) for _ in range(H)]

    def blit(grid, x0, y0, scale, colour):
        for gy, line in enumerate(grid):
            for gx, ch in enumerate(line):
                c = colour(ch)
                if c is None:
                    continue
                for dy in range(scale):
                    y = y0 + gy * scale + dy
                    if 0 <= y < H:
                        for dx in range(scale):
                            x = x0 + gx * scale + dx
                            if 0 <= x < W:
                                canvas[y][x * 4 : x * 4 + 4] = bytes(c)

    art, art_w = wordmark()
    panda_scale, word_scale, gap = 9, 26, 46
    panda_px = len(PANDA) * panda_scale
    word_px_w, word_px_h = art_w * word_scale, len(art) * word_scale
    top = (H - (panda_px + gap + word_px_h)) // 2

    blit(PANDA, (W - panda_px) // 2, top, panda_scale, PAL.get)
    blit(
        art,
        (W - word_px_w) // 2,
        top + panda_px + gap,
        word_scale,
        lambda ch: PINK if ch == "█" else None,
    )
    return write_png(path, W, H, canvas)


if __name__ == "__main__":
    for name, scale, pad in (("favicon.png", 2, 0), ("apple-touch-icon.png", 11, 2)):
        size, rows = icon(scale, pad)
        print(f"{name}: {size}x{size}, {write_png(f'src/public/{name}', size, size, rows)} bytes")
    with open("src/public/favicon.svg", "w") as f:
        f.write(svg())
    print("favicon.svg written")
    print(f"og.png: 1200x630, {preview('src/public/og.png') / 1024:.1f} KB")
