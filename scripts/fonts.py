#!/usr/bin/env python3
"""Generate the embedded fonts from the original TTF files and application text."""

import gzip
from io import BytesIO
from pathlib import Path

from fontTools import subset


ROOT = Path(__file__).resolve().parent.parent
FONTS = ROOT / "internal" / "desktop" / "fonts"


def characters():
    text = set(map(chr, range(128)))
    for first in range(0xA1, 0xF8):
        for second in range(0xA1, 0xFF):
            text.update(bytes((first, second)).decode("gb2312", errors="ignore"))

    for directory in (ROOT / "internal", ROOT / "client"):
        for path in sorted(directory.rglob("*.go")):
            text.update(path.read_text(encoding="utf-8"))

    return "".join(sorted(text))


def write_font(name, data):
    path = FONTS / f"{name}.ttf.gz"
    with BytesIO() as output:
        with gzip.GzipFile(fileobj=output, mode="wb", filename="", mtime=0) as archive:
            archive.write(data)
        path.write_bytes(output.getvalue())
    print(f"{path.relative_to(ROOT)}: {path.stat().st_size:,} bytes")


def main():
    write_font("Manrope", (FONTS / "Manrope.ttf").read_bytes())

    options = subset.Options()
    options.layout_features = ["*"]
    options.name_IDs = ["*"]
    options.name_languages = ["*"]
    options.recalc_timestamp = False

    font = subset.load_font(str(FONTS / "NotoSansSC.ttf"), options)
    subsetter = subset.Subsetter(options=options)
    subsetter.populate(text=characters())
    subsetter.subset(font)
    with BytesIO() as output:
        font.save(output)
        write_font("NotoSansSC", output.getvalue())
    font.close()


if __name__ == "__main__":
    main()
