# Icon

The extension icon (`../icon.png`, 128×128) is generated reproducibly from
[`generate_icon.py`](./generate_icon.py) — a salt shaker sprinkling grains on a
teal→blue gradient.

## Regenerate

Requires [Pillow](https://python-pillow.org/):

```bash
python3 generate_icon.py   # writes ../icon.png
```

The generator draws at 4× (512px) for anti-aliasing, then downscales to 128px.
This `icon/` directory is excluded from the packaged `.vsix` (see
`.vscodeignore`); only the resulting `icon.png` ships.
