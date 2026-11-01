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
