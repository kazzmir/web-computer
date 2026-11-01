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

Run the liquid animation (any key or mouse click exits):

```sh
go run ./cmd/web liquid.py
```

Run the GL cube (`f` toggles fullscreen; Escape exits):

```sh
go run ./cmd/web glcube.py
```

### GL compatibility

`OpenGL.GL` and `OpenGL.GLU` are emulated through Ebitengine, not PyOpenGL or
native GL calls. Both glcube paths work: indexed VBO/VAO rendering and legacy
`glBegin`/`glEnd` triangles, quads, and lines. CPU matrix transforms and
homogeneous clipping feed a Kage GPU rasterizer with per-pixel depth testing,
perspective-correct vertex colors, alpha blending, polygon offset, and line
smoothing.

This is a deliberately limited compatibility API, **not a full GLSL compiler**.
Supported vertex shaders assign `gl_Position` from
`projection * view * model * vec4(vertex_position, 1.0)` (or the untransformed
position), and `vertex_color_out` from
`(colour_mul * vertex_colour) + colour_add` (or `vertex_colour` directly).
The fragment shader assigns `fragColor = vertex_color_out`. Unsupported shader
bodies fail compilation with an info log. Up to 28 clipped primitives can be
recorded between full clears, keeping uniforms within minimum WebGL 2 budgets
(the cube uses 24).

Bundled `numpy` compatibility provides the example's numeric/structured arrays,
byte buffers, transpose, matrix multiplication, `eye`, and `zeros` only.
`ctypes.c_void_p` represents safe buffer offsets; it does not expose native memory
or FFI. No external Python packages are required.

### Runtime and tests

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
