// Go/Ebitengine registers callbacks during package initialization. Detach them
// on exit while retaining the last canvas: callbacks cannot reenter an exited
// Go runtime (e.g. a later mouse move or resize on the finished result).
export function runtimeLifecycle() {
  const listeners = [];
  const timeouts = new Set();
  const intervals = new Set();
  const frames = new Set();
  const observers = new Set();
  const add = EventTarget.prototype.addEventListener;
  const timeout = window.setTimeout.bind(window);
  const clearTimeout = window.clearTimeout.bind(window);
  const interval = window.setInterval.bind(window);
  const clearInterval = window.clearInterval.bind(window);
  const raf = window.requestAnimationFrame.bind(window);
  const cancelRaf = window.cancelAnimationFrame.bind(window);
  let closed = false;
  if (window.ResizeObserver) {
    const ResizeObserver = window.ResizeObserver;
    window.ResizeObserver = class extends ResizeObserver {
      constructor(callback) {
        super((entries, observer) => { if (!closed) callback(entries, observer); });
        observers.add(this);
      }
      disconnect() { observers.delete(this); super.disconnect(); }
    };
  }
  EventTarget.prototype.addEventListener = function (type, listener, options) {
    if (!closed) { listeners.push([this, type, listener, options]); add.call(this, type, listener, options); }
  };
  window.setTimeout = (callback, delay, ...args) => {
    const id = timeout(() => { timeouts.delete(id); if (!closed) callback(...args); }, delay);
    timeouts.add(id);
    return id;
  };
  window.clearTimeout = (id) => { timeouts.delete(id); clearTimeout(id); };
  window.setInterval = (callback, delay, ...args) => {
    const id = interval(() => { if (!closed) callback(...args); }, delay);
    intervals.add(id);
    return id;
  };
  window.clearInterval = (id) => { intervals.delete(id); clearInterval(id); };
  window.requestAnimationFrame = (callback) => {
    const id = raf((time) => { frames.delete(id); if (!closed) callback(time); });
    frames.add(id);
    return id;
  };
  window.cancelAnimationFrame = (id) => { frames.delete(id); cancelRaf(id); };
  return () => {
    closed = true;
    for (const [target, type, listener, options] of listeners) target.removeEventListener(type, listener, options);
    listeners.length = 0;
    for (const observer of observers) observer.disconnect();
    for (const id of timeouts) clearTimeout(id);
    for (const id of intervals) clearInterval(id);
    for (const id of frames) cancelRaf(id);
    timeouts.clear(); intervals.clear(); frames.clear();
  };
}
