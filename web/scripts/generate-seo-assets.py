#!/usr/bin/env python3
from __future__ import annotations

from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


ROOT = Path(__file__).resolve().parents[1]
PUBLIC = ROOT / "public"

FONT_REGULAR = "/System/Library/Fonts/Supplemental/Arial.ttf"
FONT_BOLD = "/System/Library/Fonts/Supplemental/Arial Bold.ttf"


def font(size: int, bold: bool = False) -> ImageFont.FreeTypeFont:
    return ImageFont.truetype(FONT_BOLD if bold else FONT_REGULAR, size=size)


def text_size(draw: ImageDraw.ImageDraw, text: str, text_font: ImageFont.FreeTypeFont) -> tuple[int, int]:
    left, top, right, bottom = draw.textbbox((0, 0), text, font=text_font)
    return right - left, bottom - top


def vertical_gradient(size: tuple[int, int], top: tuple[int, int, int], bottom: tuple[int, int, int]) -> Image.Image:
    width, height = size
    image = Image.new("RGB", size, top)
    pixels = image.load()
    for y in range(height):
      t = y / max(height - 1, 1)
      color = tuple(round(top[i] * (1 - t) + bottom[i] * t) for i in range(3))
      for x in range(width):
        pixels[x, y] = color
    return image


def draw_monogram(draw: ImageDraw.ImageDraw, box: tuple[int, int, int, int], scale: float = 1.0) -> None:
    x1, y1, x2, y2 = box
    width = x2 - x1
    height = y2 - y1

    radius = int(width * 0.16)
    draw.rounded_rectangle(box, radius=radius, fill="#07090d", outline="#1f2937", width=max(2, int(width * 0.018)))

    for i, color in enumerate(("#2563eb", "#10b981")):
        inset = int(width * (0.075 + i * 0.045))
        draw.rounded_rectangle(
            (x1 + inset, y1 + inset, x2 - inset, y2 - inset),
            radius=max(4, radius - inset),
            outline=color,
            width=max(4, int(width * 0.028)),
        )

    mark_font = font(max(14, int(width * 0.39 * scale)), bold=True)
    text = "GP"
    tw, th = text_size(draw, text, mark_font)
    tx = x1 + (width - tw) / 2
    ty = y1 + (height - th) / 2 - height * 0.03
    draw.text((tx + width * 0.018, ty + height * 0.018), text, font=mark_font, fill="#2563eb")
    draw.text((tx - width * 0.012, ty - height * 0.012), text, font=mark_font, fill="#f8fafc")

    cell = max(2, int(width * 0.045))
    pixel_positions = [
        (0.20, 0.23, "#10b981"),
        (0.26, 0.23, "#10b981"),
        (0.76, 0.73, "#2563eb"),
        (0.82, 0.73, "#2563eb"),
        (0.20, 0.76, "#2563eb"),
        (0.80, 0.23, "#10b981"),
    ]
    for px, py, color in pixel_positions:
        cx = x1 + int(width * px)
        cy = y1 + int(height * py)
        draw.rectangle((cx, cy, cx + cell, cy + cell), fill=color)


def create_icon(size: int) -> Image.Image:
    img = vertical_gradient((size, size), (8, 10, 15), (2, 6, 11)).convert("RGBA")
    draw = ImageDraw.Draw(img)

    for idx, color in enumerate(((37, 99, 235, 28), (16, 185, 129, 24))):
        inset = int(size * (0.05 + idx * 0.045))
        draw.rounded_rectangle(
            (inset, inset, size - inset, size - inset),
            radius=int(size * 0.22),
            outline=color,
            width=max(2, int(size * 0.012)),
        )

    pad = int(size * 0.12)
    draw_monogram(draw, (pad, pad, size - pad, size - pad))
    return img


def create_og_image() -> Image.Image:
    width, height = 1200, 630
    img = vertical_gradient((width, height), (9, 11, 18), (4, 9, 15)).convert("RGBA")
    draw = ImageDraw.Draw(img)

    # Subtle pixel grid for image-processing context.
    for x in range(0, width, 40):
        color = (255, 255, 255, 10 if x % 80 == 0 else 6)
        draw.line((x, 0, x, height), fill=color)
    for y in range(0, height, 40):
        color = (255, 255, 255, 10 if y % 80 == 0 else 6)
        draw.line((0, y, width, y), fill=color)

    draw.rounded_rectangle((70, 72, 1130, 558), radius=44, fill=(10, 13, 21, 238), outline=(39, 47, 63, 255), width=2)

    title_font = font(104, bold=True)
    subtitle_font = font(45, bold=True)
    body_font = font(31)

    text_x = 116
    draw.text((text_x, 145), "Go-Pixo", font=title_font, fill="#f8fafc")
    draw.text((text_x + 4, 272), "Fast local image compression", font=subtitle_font, fill="#60a5fa")
    draw.text((text_x + 4, 344), "PNG, JPEG, WebP, and AVIF output", font=body_font, fill="#d1d5db")
    draw.text((text_x + 4, 392), "Runs in your browser. No uploads.", font=body_font, fill="#a7f3d0")

    # Large brand mark on the right. It is intentionally separate from the text
    # so social-preview crops keep the headline readable.
    draw_monogram(draw, (790, 128, 1046, 384), scale=0.88)

    # Small compression cue under the mark.
    tile_color = (17, 24, 39, 255)
    draw.rounded_rectangle((790, 424, 884, 494), radius=14, fill=tile_color, outline=(60, 74, 94, 255), width=2)
    draw.rectangle((804, 438, 870, 459), fill=(37, 99, 235, 220))
    draw.rectangle((804, 468, 850, 481), fill=(16, 185, 129, 205))
    draw.line((910, 460, 962, 460), fill=(148, 163, 184, 230), width=5)
    draw.polygon(((962, 460), (944, 448), (944, 472)), fill=(148, 163, 184, 230))
    draw.rounded_rectangle((988, 424, 1050, 494), radius=14, fill=(7, 9, 13, 255), outline=(16, 185, 129, 255), width=3)
    for i in range(3):
        y = 444 + i * 16
        draw.line((1002, y, 1036 - i * 9, y), fill=(37, 99, 235, 220), width=6)

    # Bottom capability chips.
    chips = ["Go + WASM", "Private", "Open source"]
    x = 116
    for idx, chip in enumerate(chips):
        chip_font = font(24, bold=True)
        tw, th = text_size(draw, chip, chip_font)
        chip_box = (x, 484, x + tw + 42, 532)
        outline = "#2563eb" if idx % 2 == 0 else "#10b981"
        draw.rounded_rectangle(chip_box, radius=24, fill=(15, 23, 42, 255), outline=outline, width=2)
        draw.text((x + 21, 493), chip, font=chip_font, fill="#f8fafc")
        x += tw + 62

    return img.convert("RGB")


def write_svg() -> None:
    svg = """<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128" role="img" aria-labelledby="title desc">
  <title id="title">Go-Pixo favicon</title>
  <desc id="desc">A pixel-inspired GP mark for Go-Pixo image compression.</desc>
  <rect width="128" height="128" rx="26" fill="#07090d"/>
  <rect x="8" y="8" width="112" height="112" rx="22" fill="none" stroke="#2563eb" stroke-width="8"/>
  <rect x="18" y="18" width="92" height="92" rx="16" fill="none" stroke="#10b981" stroke-width="6"/>
  <text x="64" y="76" text-anchor="middle" font-family="Arial, Helvetica, sans-serif" font-size="47" font-weight="800" letter-spacing="-2" fill="#f8fafc">GP</text>
  <rect x="28" y="28" width="8" height="8" fill="#10b981"/>
  <rect x="38" y="28" width="8" height="8" fill="#10b981"/>
  <rect x="94" y="28" width="8" height="8" fill="#10b981"/>
  <rect x="28" y="93" width="8" height="8" fill="#2563eb"/>
  <rect x="91" y="91" width="8" height="8" fill="#2563eb"/>
</svg>
"""
    (PUBLIC / "favicon.svg").write_text(svg, encoding="utf-8")


def main() -> None:
    PUBLIC.mkdir(parents=True, exist_ok=True)
    write_svg()

    base_icon = create_icon(1024)
    for name, size in (
        ("favicon-16.png", 16),
        ("favicon-32.png", 32),
        ("apple-touch-icon.png", 180),
        ("icon-192.png", 192),
        ("icon-512.png", 512),
    ):
        resized = base_icon.resize((size, size), Image.Resampling.LANCZOS).convert("RGB")
        resized.save(PUBLIC / name, optimize=True)

    base_icon.convert("RGBA").save(PUBLIC / "favicon.ico", sizes=[(16, 16), (32, 32), (48, 48)])

    create_og_image().save(PUBLIC / "og-image.png", optimize=True)


if __name__ == "__main__":
    main()
