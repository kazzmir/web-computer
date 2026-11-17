# Browser playground

Build using the same Go installation for the application and JS runtime:

```sh
./web/build.sh
python3 -m http.server 8000 --directory web
```

Open <http://localhost:8000>. Python code is on the left; the Ebitengine canvas
and stdout/stderr console are on the right. Click **Run** (or Ctrl/Cmd+Enter).
Click the result canvas for keyboard controls. **Stop** destroys that run; the
next Run always gets a fresh iframe/Go runtime, since Ebitengine cannot restart
in the same runtime. The editor and selected example are saved in this browser's local storage.

Drag the vertical divider between the editor and results to resize both panes.
The split is saved locally. Focus the divider and use Left/Right arrows (Shift for
larger steps), or Home/End for minimum pane widths. Double-click to reset it.
On narrow screens the panes remain stacked without a divider.

Use **Syntax highlighting** above the editor to toggle Python token colors for
keywords, identifiers, function names, strings, numbers, comments, and operators.
The preference is saved locally. Unchanged tokens before and after an edit keep
their colors; only the modified text is shown as plain text immediately. After a
200 ms typing pause, the lexer restarts at a safe token boundary and corrects any
affected colors, including context changes from edited quotes or comments.
Unchanged token spans are reused. This is lightweight lexical highlighting, not
syntax validation.

In the editor, **Enter** preserves the current line's indentation and adds four
spaces after a block header ending in `:` (such as `while`, `if`, or `def`).
**Tab** inserts four spaces; Ctrl/Cmd+Enter still runs without inserting a newline.

Hover over a known function identifier to see its signature and documentation.
This works with syntax highlighting on or off. Supported names are documented
Python built-in functions and unique, undecorated top-level `def`/`async def`
functions in this file; their first-statement docstrings are shown when present.
Pygame API functions are also documented, including image loading, transforms,
display, events, mouse/keyboard input, timing, drawing, audio, and sprite collisions.
Explicit top-level imports and aliases are recognized: `import pygame as pg`,
`from pygame import image`, or `from pygame.transform import flip as mirror`.
Hover the function name in `pg.transform.flip(...)` or `pg.image.load(...)` to
see the signature supported by this browser runtime. Unknown attributes and
reassigned or shadowed aliases receive no guessed documentation.
The static index is debounced while editing. Hover lookup uses the exact lexical
identifier, never its enclosing function. Calls inside function bodies can resolve
unique functions declared later in the file. Other unknown imported functions,
inferred instance methods, unresolved module-level forward references, and names
with ambiguous or shadowed bindings are deliberately omitted rather than guessed.
Wildcard imports and dynamic namespace operations suppress the index. Tooltips never execute code and disappear on edits,
scrolling, or Escape.

Double-click a resolved user-defined function identifier to move the cursor to its
definition and scroll it into view. This also works before the documentation
index's debounce finishes. Built-ins, Pygame functions, and unresolved names keep
the usual double-click word selection.

### Debugging

Click a line number in the editor gutter to toggle a red breakpoint, then click
**Debug**. Execution pauses **before** an executable breakpoint line runs; blank
lines and comments do not trigger pauses. The debugger panel shows the current
function and its local variables. **Continue** runs to the next breakpoint,
**Step into** advances to the next user-code line (entering Python calls), and
**Step over** executes calls without stepping into them. Breakpoints inside calls
still take precedence. Imported library internals and native calls are skipped.
**Stop** discards the run, even while paused. Ordinary **Run** ignores breakpoints.
Source editing and example selection are locked during debugging so line numbers
stay aligned; breakpoints can still be toggled. Click a local variable while paused
to edit it, then press Enter or **Save**. Escape or **Cancel** discards the edit.
Supported values include strings (`"sdf"` or a bare word), integers (`3`), floats
(`8.2`), lists (`[1, 2, 3]`), dictionaries (`{1: "a"}`), `True`, `False`, and `None`.
Nested lists/dictionaries are supported. Edits update the real local variable,
including function arguments and closure cells, without resuming execution.
Invalid values show an error and leave the variable unchanged; calls and other
expressions are not evaluated. Values are displayed safely without invoking
user-defined `__repr__`; complex objects show their type, and lists/dictionaries
are bounded previews. Breakpoints are session-only and reset on example changes.

Optional browser regression tests: serve the built app, install Playwright, then
run `node web/debug-test.mjs`, `node web/splitter-test.mjs`, or
`node web/highlight-test.mjs`, `node web/editor-input-test.mjs`, or
`node web/hover-test.mjs` (supports `BASE_URL` and `PLAYWRIGHT_MODULE`).
Static function-resolution tests also run without a browser:
`node web/function-index-test.mjs`.

The **Example** dropdown includes Chimp, Aliens, Liquid, Scroll, Scaletest,
GL cube, and the SDL2 Video demo, alongside the welcome animation. Selecting
one replaces the editor with its unchanged source; press Run to execute it.
Reset example reloads the selected demo. Scaletest automatically receives
`data/chimp.png` as an argument. Example filenames are preserved for tracebacks
and `__file__`-relative asset loading; assets remain embedded in the WASM build.
Click the canvas to enable keyboard/audio interaction. Some demos request
fullscreen; press Escape to leave fullscreen and access the editor controls.

`wasm_exec.js` is Go's standard runtime support (license in `LICENSE-Go`).
`build.sh` refreshes it from `go env GOROOT` to match the compiler. The generated
`web-computer.wasm` is ignored by Git: rebuild it after changing Go/Python code.
The build also copies `scripts/python/*.py` into the ignored `web/examples/`
directory, so there is no second checked-in copy of the example sources.
Serve over HTTP, not `file://`; no npm packages or CDN are required.

The runner sends editor text as data to the WASM application, not as HTML or
JavaScript. The JS build reads that text as `editor.py`; native CLI scripts and
embedded example assets still work as before. Python errors appear in the
console and set an error status. A normal exit leaves the last canvas visible.

This is the project's limited Python/pygame runtime, not full CPython or a
security sandbox for untrusted programs. Browser execution periodically yields
between Python instructions, so ordinary tight Python loops can be stopped.
Very expensive native operations, compilation, and huge allocations can still
block or exhaust the tab. Stop/re-run discards interpreter state immediately;
it does not guarantee execution of Python `finally` blocks.

## GitHub Pages

The [Pages workflow](../.github/workflows/pages.yml) builds and publishes the
playground on pushes to `master` or `main`, or via **Actions → Publish playground
to GitHub Pages → Run workflow**.

In the repository, first choose **Settings → Pages → Build and deployment →
Source: GitHub Actions**. The workflow uses Go from `go.mod`, builds the WASM
binary, refreshes the matching `wasm_exec.js`, and copies the example sources.
It uploads only the website assets as a Pages artifact, then deploys using the
built-in GitHub token/OIDC; no custom secrets or `gh-pages` branch are needed.
The generated WASM binary is never committed to Git.

The deployed URL appears in the workflow's `github-pages` environment (normally
`https://<owner>.github.io/<repository>/`). All asset URLs are relative, so
repository subpaths work without configuration.

## Optional browser smoke test

With the HTTP server running, install Playwright just for testing:

```sh
cd web
npm install --no-save --package-lock=false playwright
npx playwright install chromium
node smoke-test.mjs
```

Set `BASE_URL` for another server address. The test checks actual animated
canvas output, UTF-8 stdout, syntax/runtime errors, isolated reruns, stopping a
tight loop, saved editor contents, clean runtime shutdown, and all seven bundled
examples (including filenames, scaler arguments, saved selection, and reset).
