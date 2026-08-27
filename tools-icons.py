import zlib, struct

PINK  = (0xff, 0x87, 0xd7, 255)
INK   = (0x0b, 0x0b, 0x0f, 255)
SNOW  = (0xf2, 0xf2, 0xf5, 255)
PAL = {"P": PINK, "K": INK, "W": SNOW}

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

# A face that is a pixel off-centre is obvious at any size.
assert all(len(r) == 16 for r in PANDA), "grid must be 16 wide"
assert all(r == r[::-1] for r in PANDA), "grid must be symmetric"

def write_png(path, w, h, pixels):
    raw = b"".join(b"\x00" + bytes(pixels[y]) for y in range(h))
    def chunk(tag, data):
        return (struct.pack(">I", len(data)) + tag + data
                + struct.pack(">I", zlib.crc32(tag + data) & 0xFFFFFFFF))
    png = (b"\x89PNG\r\n\x1a\n"
           + chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 6, 0, 0, 0))
           + chunk(b"IDAT", zlib.compress(raw, 9))
           + chunk(b"IEND", b""))
    open(path, "wb").write(png)
    return len(png)

def render(grid, scale, pad=0, bg=PINK):
    gh, gw = len(grid), len(grid[0])
    w, h = gw * scale + pad * 2, gh * scale + pad * 2
    rows = []
    for y in range(h):
        row = bytearray()
        gy = (y - pad) // scale
        for x in range(w):
            gx = (x - pad) // scale
            if 0 <= gy < gh and 0 <= gx < gw:
                row += bytes(PAL.get(grid[gy][gx], bg))
            else:
                row += bytes(bg)
        rows.append(row)
    return w, h, rows

for name, scale, pad in (("favicon.png", 2, 0), ("apple-touch-icon.png", 11, 2)):
    w, h, rows = render(PANDA, scale, pad)
    n = write_png(f"src/public/{name}", w, h, rows)
    print(f"{name}: {w}x{h}, {n} bytes")

# SVG: same grid, one rect per run of colour, so it stays tiny and crisp.
out = ['<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" shape-rendering="crispEdges">',
       '<rect width="16" height="16" fill="#ff87d7"/>']
for y, line in enumerate(PANDA):
    x = 0
    while x < len(line):
        c = line[x]
        run = 1
        while x + run < len(line) and line[x + run] == c:
            run += 1
        if c in ("K", "W"):
            fill = "#0b0b0f" if c == "K" else "#f2f2f5"
            out.append(f'<rect x="{x}" y="{y}" width="{run}" height="1" fill="{fill}"/>')
        x += run
out.append("</svg>")
open("src/public/favicon.svg", "w").write("\n".join(out))
print("favicon.svg:", len("\n".join(out)), "bytes")
