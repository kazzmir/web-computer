Pygame implementation in Golang backed by Ebitengine.

Run the bundled Chimp example (Escape exits):

```sh
go run ./cmd/web chimp.py
```

Scripts and their `data/` assets are embedded from `scripts/python/`. Images,
fonts, and WAV sounds can also be loaded from filesystem paths. The pygame
subset includes surfaces, rectangles, image transforms, sprite groups, mouse
and keyboard input, text rendering, sound playback, and frame timing.

Tests require a display; on headless Linux use:

```sh
xvfb-run -a go test ./cmd/...
go test ./gpython/stdlib/os ./gpython/stdlib/builtin ./gpython/py
```
