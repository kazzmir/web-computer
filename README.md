Pygame implementation in Golang backed by Ebitengine.

Run the bundled Chimp example (Escape exits):

```sh
go run ./cmd/web chimp.py
```

Run the interactive scaler (arrow keys resize; Escape exits):

```sh
go run ./cmd/web scaletest.py data/chimp.png
# Optional benchmark and alpha conversion:
go run ./cmd/web scaletest.py data/chimp.png -t -convert_alpha
```

Run the zoomed scroll viewer (arrow keys or click the triangles; Escape exits):

```sh
go run ./cmd/web scroll.py
# Or supply an image:
go run ./cmd/web scroll.py data/chimp.png
```

The embedded runtime supports f-strings, including expressions, `!s`/`!r`/`!a`
conversions, escaped braces, and numeric format specs such as `.4f`.

Scripts and their `data/` assets are embedded from `scripts/python/`. Images,
fonts, and WAV sounds can also be loaded from filesystem paths. The pygame
subset includes surfaces, rectangles, image transforms, sprite groups, mouse
and keyboard input, text rendering, sound playback, and frame timing.

Tests require a display; on headless Linux use:

```sh
xvfb-run -a go test ./cmd/...
go test ./gpython/stdlib/os ./gpython/stdlib/builtin ./gpython/py
```
