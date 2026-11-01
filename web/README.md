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
in the same runtime. The editor is saved in this browser's local storage.

`wasm_exec.js` is Go's standard runtime support (license in `LICENSE-Go`).
`build.sh` refreshes it from `go env GOROOT` to match the compiler. The generated
`web-computer.wasm` is ignored by Git: rebuild it after changing Go/Python code.
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
tight loop, saved editor contents, and clean runtime shutdown.
