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
On narrow screens the editor and results remain stacked without that divider.

Drag the horizontal dividers in the results pane to distribute vertical space
between the runtime, debugger, and console. Each divider resizes its adjacent
panes without restarting the program; debugger locals and call stack fill their
allocated space and scroll. With the debugger hidden, the remaining divider
resizes runtime versus console. Normal/debug layouts have separate saved sizes.
Focus a horizontal divider and use Up/Down (Shift for larger steps), or Home/End
to reach minimum heights. Double-click either to reset that layout. Minimums keep
the runtime, debugger controls, and console usable; very short windows can scroll.
These dividers also work on mobile and while the interpreter is paused.

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
**Tab** inserts four spaces. **Shift+Tab** removes one indentation level from the
current line or all selected lines, preserving the caret/selection. One level is
four leading spaces or a tab; shorter indentation is removed entirely. A selection
ending at the next line's start leaves that line unchanged.
Ctrl/Cmd+Enter still runs without inserting a newline.
With the editor focused, a caret on or immediately after `(`, `)`, `[`, `]`, `{`,
or `}` highlights both members of its matching pair. This works with syntax
colors on or off, handles nesting and multiple lines, and ignores strings and
comments. Unmatched/misnested delimiters and nonempty selections get no matching
highlight. This does not automatically insert closing brackets.

Syntax checks run after a 300 ms editing pause in an isolated background worker,
using the same Python compiler as Run without executing the program. The first
syntax error is marked with a red warning triangle, tinted background, and edge
stripe on its gutter line (separate from the breakpoint dot). Hover that line number for the error, or read its line, column, and message
below the editor. Breakpoint dots still work on error lines. Fixing or changing
the code clears stale markers immediately; old background results are discarded.
Checks also catch compiler errors such as `return` outside a function. They do
not report runtime errors or undefined names. If the checker cannot load, editing
and Run still work; rebuild with `./web/build.sh` to regenerate its WASM asset.

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

Double-click a resolved user-defined function identifier or a constructor call to
a unique, undecorated top-level class to move the cursor to its definition and
scroll it into view. Class constructors jump to the class name, not `__init__`.
This also works before the documentation index's debounce finishes. Built-in and
Pygame functions/constructors, and unresolved or shadowed names, keep the usual
double-click word selection.

### Debugging

Click a line number in the editor gutter to toggle a red breakpoint, then click
**Debug**. Execution pauses **before** an executable breakpoint line runs; blank
lines and comments do not trigger pauses. The debugger inspector is split into
**Call stack** and **Locals** panes. The stack lists active Python frames, newest
first, with function names and filename/line locations. Select a frame to inspect
its locals; the arrow marks the actual executing frame. Selecting a frame from the
current source also moves the editor caret to its line and scrolls it into view;
a blue gutter highlight distinguishes an inspected caller from the stopped line.
Caller locations show the call site, and recursive calls remain separate frames.
Frames from other files still show locals but do not jump into unrelated editor
text. Selecting a frame never resumes or moves execution. Each new pause selects
the executing frame again.
**Continue** runs to the next breakpoint,
**Step into** advances to the next user-code line (entering Python calls), and
**Step over** executes calls without stepping into them. **Step out** runs until
the current function returns, then pauses at the next user-code line in its caller.
It is disabled outside functions (including module and class bodies), even after
editing variables; the interpreter also rejects such requests. Recursive calls
step out one level at a time. Breakpoints still take precedence over stepping.
Imported library internals and native calls are skipped by ordinary stepping.
**Break** requests a temporary pause at the next Python line, including library
code, without adding a breakpoint. It is enabled only while the debug interpreter
is ready and running, and disabled while paused, starting, or finished. A native
call must return/yield to another Python line before it can stop. Library pauses
show their actual filename and stack, without highlighting an unrelated editor
line. The request is consumed by that pause; Continue resumes normally.
**Stop** discards the run, even while paused. Ordinary **Run** ignores breakpoints.
Stepping always controls the currently executing frame, even when the module or
a caller is selected for inspection. Native Go calls do not add Python stack rows.
Source editing and example selection are locked during debugging so line numbers
stay aligned; breakpoints can still be toggled. Click a local variable while paused
to edit it, then press Enter or **Save**. Escape or **Cancel** discards the edit.
Supported values include strings (`"sdf"` or a bare word), integers (`3`), floats
(`8.2`), lists (`[1, 2, 3]`), dictionaries (`{1: "a"}`), `True`, `False`, and `None`.
Nested lists/dictionaries are supported. Edits update the real local variable in
the selected stack frame, including caller locals, module globals, function
arguments, and closure cells, without resuming execution. Frame switching and
stepping are disabled until the edit is saved or cancelled.
Invalid values show an error and leave the variable unchanged; calls and other
expressions are not evaluated. Values are displayed safely without invoking
user-defined `__repr__`; complex objects show their type, and lists/dictionaries
are bounded previews. Breakpoints are session-only and reset on example changes.

Optional browser regression tests: serve the built app, install Playwright, then
run `node web/debug-test.mjs`, `node web/splitter-test.mjs`, or
`node web/highlight-test.mjs`, `node web/editor-input-test.mjs`, or
`node web/hover-test.mjs`, `node web/syntax-check-test.mjs`, or
`node web/brackets-test.mjs`, `node web/step-out-test.mjs`, or
`node web/call-stack-test.mjs`, `node web/result-splitters-test.mjs`, or
`node web/break-test.mjs` (supports `BASE_URL` and `PLAYWRIGHT_MODULE`).
Static function-resolution tests also run without a browser:
`node web/function-index-test.mjs`, `node web/bracket-pairs-test.mjs`, and
`node web/dedent-test.mjs`.

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
`web-computer.wasm` and the background compiler's `syntax-check.wasm` are ignored
by Git: rebuild them after changing Go/Python code.
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
