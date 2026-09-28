#!/usr/bin/env python3
"""Refresh Scout's searchable Dashboard Icons index from the upstream repository."""
import json
from pathlib import Path
from urllib.request import urlopen

BASE = "https://raw.githubusercontent.com/homarr-labs/dashboard-icons/main/"
with urlopen(BASE + "metadata.json") as response:
    metadata = json.load(response)
with urlopen(BASE + "tree.json") as response:
    tree = json.load(response)

available = {format: {name.removesuffix("." + format) for name in files}
             for format, files in tree.items() if format in ("svg", "png", "webp")}
icons = []
for slug, info in sorted(metadata.items()):
    format = next((kind for kind in ("svg", "webp", "png") if slug in available.get(kind, set())), None)
    if format:
        aliases = [alias for alias in info.get("aliases", []) if isinstance(alias, str)]
        icons.append({"slug": slug, "format": format, "aliases": aliases,
                      "light": slug + "-light" in available[format]})

output = Path(__file__).resolve().parents[1] / "web/src/lib/dashboard-icons.json"
output.write_text(json.dumps(icons, ensure_ascii=False, separators=(",", ":")) + "\n")
print(f"Wrote {len(icons)} icons to {output}")
