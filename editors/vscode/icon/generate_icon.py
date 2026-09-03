#!/usr/bin/env python3
"""Generate the Salty VS Code extension icon: a salt shaker sprinkling grains.

Draws at 4x (512px) for anti-aliasing, then downscales to the 128x128 PNG that
VS Code expects. Run from this directory:

    python3 generate_icon.py
"""

import os

from PIL import Image, ImageDraw

SCALE = 4
SIZE = 128 * SCALE


def lerp(a, b, t):
    return tuple(round(a[i] + (b[i] - a[i]) * t) for i in range(3))


def rounded_rect_mask(size, radius):
    mask = Image.new("L", (size, size), 0)
    d = ImageDraw.Draw(mask)
    d.rounded_rectangle([0, 0, size - 1, size - 1], radius=radius, fill=255)
    return mask


def main():
    # Branded background: vertical teal -> deep blue gradient (matches the
    # crypto/Solidity feel), clipped to a rounded square.
    top = (0x2D, 0xD4, 0xBF)   # teal
    bottom = (0x14, 0x53, 0x8B)  # deep blue
    bg = Image.new("RGB", (SIZE, SIZE), bottom)
    px = bg.load()
    for y in range(SIZE):
        row = lerp(top, bottom, y / (SIZE - 1))
        for x in range(SIZE):
            px[x, y] = row

    img = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    img.paste(bg, (0, 0), rounded_rect_mask(SIZE, radius=24 * SCALE))
    d = ImageDraw.Draw(img)

    cx = SIZE // 2
    white = (0xFB, 0xFC, 0xFE)
    cap = (0xE7, 0xEC, 0xF3)
    outline = (0x0E, 0x3A, 0x63)
    ow = 3 * SCALE

    # --- Salt shaker body (rounded flask shape) ---
    body_top = 40 * SCALE
    body_bottom = 92 * SCALE
    body_left = cx - 26 * SCALE
    body_right = cx + 26 * SCALE
    d.rounded_rectangle(
        [body_left, body_top, body_right, body_bottom],
        radius=16 * SCALE, fill=white, outline=outline, width=ow,
    )

    # Shoulders taper into the cap: cover the top corners with the bg-less
    # trapezoid by drawing a narrower neck on top.
    neck_left = cx - 17 * SCALE
    neck_right = cx + 17 * SCALE
    neck_top = 30 * SCALE
    d.polygon(
        [(body_left + ow, body_top + 6 * SCALE),
         (neck_left, neck_top),
         (neck_right, neck_top),
         (body_right - ow, body_top + 6 * SCALE)],
        fill=white, outline=outline,
    )

    # --- Cap (perforated top) ---
    cap_top = 20 * SCALE
    cap_bottom = 33 * SCALE
    d.rounded_rectangle(
        [neck_left - 2 * SCALE, cap_top, neck_right + 2 * SCALE, cap_bottom],
        radius=5 * SCALE, fill=cap, outline=outline, width=ow,
    )
    # Perforations in the cap.
    for i in (-1, 0, 1):
        hx = cx + i * 8 * SCALE
        hy = (cap_top + cap_bottom) // 2
        d.ellipse([hx - 2 * SCALE, hy - 2 * SCALE, hx + 2 * SCALE, hy + 2 * SCALE],
                  fill=outline)

    # --- "S" monogram on the body ---
    try:
        from PIL import ImageFont
        font = ImageFont.truetype(
            "/System/Library/Fonts/Helvetica.ttc", 40 * SCALE)
    except Exception:
        font = None
    label_y = 64 * SCALE
    if font is not None:
        d.text((cx, label_y), "S", font=font, fill=(0x14, 0x53, 0x8B),
               anchor="mm")
    else:
        d.text((cx, label_y), "S", fill=(0x14, 0x53, 0x8B), anchor="mm")

    # --- Sprinkled salt grains (small diamonds) below the shaker ---
    grains = [
        (cx - 22 * SCALE, 104 * SCALE, 3.5),
        (cx - 6 * SCALE, 112 * SCALE, 4.0),
        (cx + 12 * SCALE, 106 * SCALE, 3.0),
        (cx + 24 * SCALE, 116 * SCALE, 3.5),
        (cx - 14 * SCALE, 120 * SCALE, 2.5),
        (cx + 2 * SCALE, 122 * SCALE, 3.0),
    ]
    for gx, gy, r in grains:
        r = r * SCALE
        d.polygon([(gx, gy - r), (gx + r, gy), (gx, gy + r), (gx - r, gy)],
                  fill=white)

    out = img.resize((128, 128), Image.LANCZOS)
    # Write to the extension root, where package.json's "icon" field points.
    dest = os.path.join(os.path.dirname(__file__), "..", "icon.png")
    out.save(dest)
    print("wrote", os.path.normpath(dest), "(128x128)")


if __name__ == "__main__":
    main()
