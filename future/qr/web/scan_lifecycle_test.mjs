// Run with: node --experimental-vm-modules --test web/scan_lifecycle_test.mjs
// The application script is evaluated unchanged. DOM/media promises are synthetic;
// the final two cases also execute the exact bundled Nimiq lifecycle implementation.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import vm from 'node:vm';

const source = readFileSync(new URL('./static/scan.js', import.meta.url), 'utf8');
const bundledScanner = readFileSync(new URL('./static/vendor/qr-scanner/qr-scanner.min.js', import.meta.url), 'utf8');
const flush = () => new Promise(resolve => setImmediate(resolve));
function deferred() {
  let resolve, reject;
  const promise = new Promise((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}

function target(properties = {}) {
  const listeners = new Map();
  return {
    hidden: false, disabled: false, value: '', textContent: '', ...properties,
    addEventListener(name, fn) {
      if (!listeners.has(name)) listeners.set(name, new Set());
      listeners.get(name).add(fn);
    },
    removeEventListener(name, fn) { listeners.get(name)?.delete(fn); },
    dispatch(name, event) { return [...(listeners.get(name) || [])].map(fn => fn(event)); },
    focus() {},
  };
}

function harness(options = {}) {
  const stats = { submissions: 0, detections: 0, imports: 0, imageScans: 0, scanners: [], streams: [] };
  const timers = new Map();
  let timerID = 0;
  class MediaStream {
    constructor() {
      this.stops = 0;
      this.tracks = [{ label: 'environment', stop: () => { this.stops++; } }];
      stats.streams.push(this);
    }
    getTracks() { return [...this.tracks]; }
    getVideoTracks() { return this.getTracks(); }
    removeTrack(track) { this.tracks = this.tracks.filter(item => item !== track); }
  }
  const nodes = {
    'camera-section': target(), 'camera-start': target(), 'camera-stop': target(),
    'camera-file': target({ files: [] }), 'camera-status': target(), code: target(),
    'manual-form': target({
      requestSubmit() {
        const event = { defaultPrevented: false, preventDefault() { this.defaultPrevented = true; } };
        this.dispatch('submit', event);
        if (!event.defaultPrevented) stats.submissions++;
      },
      submit() { stats.submissions++; },
    }),
  };
  const body = { contains: element => element === nodes['camera-video'], appendChild() {} };
  function videoNode() {
    const video = target({ srcObject: null, style: {}, parentElement: body, paused: true,
      videoWidth: 640, videoHeight: 480 });
    video.play = () => { video.paused = false; return options.play ? options.play(video) : Promise.resolve(); };
    video.pause = () => { video.paused = true; };
    video.cloneNode = () => { const copy = videoNode(); copy.hidden = video.hidden; return copy; };
    video.replaceWith = next => { assert.equal(nodes['camera-video'], video); nodes['camera-video'] = next; };
    return video;
  }
  nodes['camera-video'] = videoNode();
  const document = target({ hidden: false, body,
    getElementById: id => nodes[id],
    createElement: name => { assert.equal(name, 'canvas'); return {}; },
  });
  class Detector {
    detect(video) { stats.detections++; return options.detect ? options.detect(video) : new Promise(() => {}); }
  }
  class Worker { postMessage() {} addEventListener() {} removeEventListener() {} terminate() {} }
  const window = target({ isSecureContext: options.secure !== false, location: { protocol: 'https:' } });
  if (options.native !== false) window.BarcodeDetector = Detector;
  class Scanner {
    constructor(video, callback) {
      this.video = video; this.callback = callback; this.stops = 0; this.destroyed = 0;
      stats.scanners.push(this);
    }
    async start() {
      if (options.libraryStart) return options.libraryStart(this, MediaStream);
      this.video.srcObject = new MediaStream();
    }
    stop() { this.stops++; }
    destroy() { this.destroyed++; }
    static scanImage(file) {
      stats.imageScans++;
      return options.scanImage ? options.scanImage(file) : Promise.resolve({ data: 'image-code' });
    }
  }
  const context = vm.createContext({ document, window, BarcodeDetector: Detector, MediaStream, Worker,
    navigator: { mediaDevices: { getUserMedia: () => options.acquire ? options.acquire(MediaStream) : Promise.resolve(new MediaStream()) } },
    setTimeout(fn, delay) { const id = ++timerID; timers.set(id, { fn, delay }); return id; },
    clearTimeout(id) { timers.delete(id); }, requestAnimationFrame: () => 0, console,
  });
  let realModule;
  async function loadBundled() {
    if (!realModule) {
      realModule = new vm.SourceTextModule(bundledScanner, {
        context, identifier: 'bundled-qr-scanner',
        importModuleDynamically: async name => {
          assert.equal(name, './qr-scanner-worker.min.js');
          const workerModule = new vm.SyntheticModule(['createWorker'], function () {
            this.setExport('createWorker', () => new Worker());
          }, { context });
          await workerModule.link(() => assert.fail('unexpected worker import'));
          await workerModule.evaluate();
          return workerModule;
        },
      });
      await realModule.link(() => assert.fail('unexpected scanner static import'));
      await realModule.evaluate();
    }
    return realModule;
  }
  new vm.Script(source, {
    filename: 'web/static/scan.js',
    importModuleDynamically: async name => {
      assert.equal(name, '/qr/static/vendor/qr-scanner/qr-scanner.min.js');
      stats.imports++;
      if (options.load) await options.load(stats.imports);
      if (options.bundled) return loadBundled();
      const module = new vm.SyntheticModule(['default'], function () { this.setExport('default', Scanner); }, { context });
      await module.link(() => assert.fail('unexpected fixture import'));
      await module.evaluate();
      return module;
    },
  }).runInContext(context, { timeout: 1000 });
  return {
    nodes, stats, timers, document, window, MediaStream,
    start: () => nodes['camera-start'].dispatch('click')[0],
    stop: () => nodes['camera-stop'].dispatch('click'),
    leave: () => window.dispatch('pagehide'),
    logout: () => window.dispatch('circula:logout'),
    hide() { document.hidden = true; document.dispatch('visibilitychange'); },
    file(file = { size: 32 }) { nodes['camera-file'].files = [file]; nodes['camera-file'].value = 'selected'; return nodes['camera-file'].dispatch('change')[0]; },
    async runTimers() {
      const scheduled = [...timers.values()]; timers.clear();
      for (const timer of scheduled) timer.fn();
      await flush();
    },
  };
}

for (const event of ['stop', 'leave', 'hide', 'logout']) {
  test(`late camera permission after ${event} releases the acquired stream`, { timeout: 2000 }, async () => {
    const permission = deferred();
    const h = harness({ acquire: () => permission.promise });
    const pending = h.start();
    h[event]();
    const stream = new h.MediaStream(); permission.resolve(stream);
    await pending;
    assert.ok(stream.stops > 0);
    assert.equal(h.nodes['camera-video'].srcObject, null);
    assert.equal(h.stats.detections, 0);
    assert.equal(h.nodes['camera-start'].hidden, false);
  });
}

test('hidden state at permission completion is checked even without a visibility event', { timeout: 2000 }, async () => {
  const permission = deferred(); const h = harness({ acquire: () => permission.promise });
  const pending = h.start(); h.document.hidden = true;
  const stream = new h.MediaStream(); permission.resolve(stream); await pending;
  assert.ok(stream.stops > 0); assert.equal(h.stats.detections, 0);
});

test('leaving during video.play cannot start detection afterwards', { timeout: 2000 }, async () => {
  const play = deferred(); const h = harness({ play: () => play.promise });
  const pending = h.start(); await flush(); h.leave(); play.resolve(); await pending;
  assert.ok(h.stats.streams[0].stops > 0); assert.equal(h.stats.detections, 0);
});

test('late permission from an earlier attempt cannot replace the new camera', { timeout: 2000 }, async () => {
  const first = deferred(); let count = 0;
  const h = harness({ acquire: Stream => ++count === 1 ? first.promise : Promise.resolve(new Stream()) });
  const old = h.start(); h.stop(); await h.start();
  const current = h.nodes['camera-video'].srcObject;
  const stale = new h.MediaStream(); first.resolve(stale); await old;
  assert.ok(stale.stops > 0); assert.equal(current.stops, 0);
  assert.equal(h.nodes['camera-video'].srcObject, current);
});

test('old camera failure cannot stop or overwrite the status of a newer attempt', { timeout: 2000 }, async () => {
  const first = deferred(); let count = 0;
  const h = harness({ acquire: Stream => ++count === 1 ? first.promise : Promise.resolve(new Stream()) });
  const old = h.start(); h.stop(); await h.start();
  const current = h.nodes['camera-video'].srcObject, status = h.nodes['camera-status'].textContent;
  first.reject(new Error('old permission refused')); await old;
  assert.equal(current.stops, 0); assert.equal(h.nodes['camera-video'].srcObject, current);
  assert.equal(h.nodes['camera-status'].textContent, status);
});

test('late native decode after stop cannot fill or submit the preview form', { timeout: 2000 }, async () => {
  const detection = deferred(); const h = harness({ detect: () => detection.promise });
  await h.start(); h.stop(); detection.resolve([{ rawValue: 'stale-code' }]); await flush();
  assert.equal(h.nodes.code.value, ''); assert.equal(h.stats.submissions, 0);
});

for (const reject of [false, true]) {
  test(`cancelled native ${reject ? 'failure' : 'empty result'} cannot schedule another scan`, { timeout: 2000 }, async () => {
    const detection = deferred(); const h = harness({ detect: () => detection.promise });
    await h.start(); h.stop();
    if (reject) detection.reject(new Error('old decode failed')); else detection.resolve([]);
    await flush(); assert.equal(h.timers.size, 0);
  });
}

test('current native result keeps the existing preview navigation and input bound', { timeout: 2000 }, async () => {
  const detection = deferred(); const h = harness({ detect: () => detection.promise });
  await h.start(); detection.resolve([{ rawValue: 'valid-code' }]); await flush();
  assert.equal(h.nodes.code.value, 'valid-code'); assert.equal(h.stats.submissions, 1);
  assert.ok(h.stats.streams[0].stops > 0);
  const oversized = deferred(); const other = harness({ detect: () => oversized.promise });
  await other.start(); oversized.resolve([{ rawValue: 'x'.repeat(2049) }]); await flush();
  assert.equal(other.nodes.code.value, ''); assert.equal(other.stats.submissions, 0);
});

test('cancelled fallback import cannot construct or start a scanner', { timeout: 2000 }, async () => {
  const imported = deferred(); const h = harness({ native: false, load: () => imported.promise });
  const pending = h.start(); await flush(); h.stop(); imported.resolve(); await pending;
  assert.equal(h.stats.scanners.length, 0); assert.equal(h.stats.streams.length, 0);
});

test('late fallback start releases only its own media', { timeout: 2000 }, async () => {
  const started = deferred(); let count = 0;
  const h = harness({ native: false, libraryStart: async (scanner, Stream) => {
    if (++count === 1) await started.promise;
    scanner.video.srcObject = new Stream();
  } });
  const old = h.start(); await flush(); h.stop(); await h.start();
  const current = h.nodes['camera-video'].srcObject;
  started.resolve(); await old;
  assert.equal(h.nodes['camera-video'].srcObject, current); assert.equal(current.stops, 0);
  assert.ok(h.stats.streams[1].stops > 0);
});

test('old fallback callback cannot consume a newer camera attempt', { timeout: 2000 }, async () => {
  const h = harness({ native: false }); await h.start(); const old = h.stats.scanners[0];
  h.stop(); await h.start(); const current = h.nodes['camera-video'].srcObject;
  old.callback({ data: 'stale-code' });
  assert.equal(h.nodes.code.value, ''); assert.equal(h.stats.submissions, 0);
  assert.equal(h.nodes['camera-video'].srcObject, current); assert.equal(current.stops, 0);
});

test('old fallback start failure cannot stop a newer scanner', { timeout: 2000 }, async () => {
  const started = deferred(); let count = 0;
  const h = harness({ native: false, libraryStart: async (scanner, Stream) => {
    if (++count === 1) await started.promise;
    scanner.video.srcObject = new Stream();
  } });
  const old = h.start(); await flush(); h.stop(); await h.start();
  const current = h.nodes['camera-video'].srcObject;
  started.reject(new Error('old start failed')); await old;
  assert.equal(h.nodes['camera-video'].srcObject, current); assert.equal(current.stops, 0);
  assert.equal(h.stats.scanners[1].destroyed, 0);
});

test('current fallback callback still navigates once to the preview', { timeout: 2000 }, async () => {
  const h = harness({ native: false }); await h.start();
  h.stats.scanners[0].callback({ data: 'fallback-code' });
  assert.equal(h.nodes.code.value, 'fallback-code'); assert.equal(h.stats.submissions, 1);
  assert.ok(h.stats.streams[0].stops > 0);
  h.stats.scanners[0].callback({ data: 'duplicate-code' });
  assert.equal(h.stats.submissions, 1);
});

test('cancelled image import cannot start decoding', { timeout: 2000 }, async () => {
  const imported = deferred(); const h = harness({ load: () => imported.promise });
  const pending = h.file(); await flush(); h.stop(); imported.resolve(); await pending;
  assert.equal(h.stats.imageScans, 0); assert.equal(h.stats.submissions, 0);
});

test('late image result after leave cannot fill or submit the form', { timeout: 2000 }, async () => {
  const decoded = deferred(); const h = harness({ scanImage: () => decoded.promise });
  const pending = h.file(); await flush(); h.leave(); decoded.resolve({ data: 'old-image' }); await pending;
  assert.equal(h.nodes.code.value, ''); assert.equal(h.stats.submissions, 0);
});

test('an earlier image result cannot supersede the latest selected file', { timeout: 2000 }, async () => {
  const first = deferred(), second = deferred(); let count = 0;
  const h = harness({ scanImage: () => ++count === 1 ? first.promise : second.promise });
  const old = h.file(); await flush(); const current = h.file(); await flush();
  first.resolve({ data: 'old-image' }); await old;
  assert.equal(h.nodes.code.value, ''); assert.equal(h.stats.submissions, 0);
  second.resolve({ data: 'current-image' }); await current;
  assert.equal(h.nodes.code.value, 'current-image'); assert.equal(h.stats.submissions, 1);
});

test('an oversized replacement image cancels the previous pending decode', { timeout: 2000 }, async () => {
  const decoded = deferred(); const h = harness({ scanImage: () => decoded.promise });
  const pending = h.file(); await flush();
  await h.file({ size: 10 * 1024 * 1024 + 1 });
  const error = h.nodes['camera-status'].textContent;
  assert.match(error, /under 10 MB/); assert.equal(h.stats.imageScans, 1);
  decoded.resolve({ data: 'stale-earlier-image' }); await pending;
  assert.equal(h.nodes.code.value, ''); assert.equal(h.stats.submissions, 0);
  assert.equal(h.nodes['camera-status'].textContent, error);
});

test('a file event without a new selection does not cancel the pending image', { timeout: 2000 }, async () => {
  const decoded = deferred(); const h = harness({ scanImage: () => decoded.promise });
  const pending = h.file(); await flush();
  h.nodes['camera-file'].files = [];
  await h.nodes['camera-file'].dispatch('change')[0];
  decoded.resolve({ data: 'original-image' }); await pending;
  assert.equal(h.nodes.code.value, 'original-image'); assert.equal(h.stats.submissions, 1);
});

test('manual form submission cancels a pending native camera result', { timeout: 2000 }, async () => {
  const detection = deferred(); const h = harness({ detect: () => detection.promise });
  await h.start();
  const stream = h.nodes['camera-video'].srcObject;
  h.nodes.code.value = 'new-manual-code'; h.nodes['manual-form'].requestSubmit();
  assert.equal(h.stats.submissions, 1);
  detection.resolve([{ rawValue: 'old-camera-code' }]); await flush();
  assert.equal(h.nodes.code.value, 'new-manual-code'); assert.equal(h.stats.submissions, 1);
  assert.ok(stream.stops > 0); assert.equal(h.nodes['camera-video'].srcObject, null);
});

test('manual form submission cancels a pending image result', { timeout: 2000 }, async () => {
  const decoded = deferred(); const h = harness({ scanImage: () => decoded.promise });
  const pending = h.file(); await flush();
  h.nodes.code.value = 'new-manual-code'; h.nodes['manual-form'].requestSubmit();
  assert.equal(h.stats.submissions, 1);
  decoded.resolve({ data: 'old-image-code' }); await pending;
  assert.equal(h.nodes.code.value, 'new-manual-code'); assert.equal(h.stats.submissions, 1);
});

test('unsupported camera retains the manual path and never requests permission', { timeout: 2000 }, async () => {
  const h = harness({ secure: false, acquire: () => assert.fail('permission requested') });
  await h.start(); assert.equal(h.stats.submissions, 0);
  assert.match(h.nodes['camera-status'].textContent, /Enter the code by hand/);
});

test('bundled scanner media is stopped immediately before its delayed cleanup', { timeout: 2000 }, async () => {
  const h = harness({ native: false, bundled: true }); await h.start();
  const stream = h.nodes['camera-video'].srcObject; assert.ok(stream);
  h.stop(); assert.ok(stream.stops > 0);
  await h.runTimers(); assert.equal(h.nodes['camera-video'].srcObject, null);
});

test('bundled scanner delayed cleanup cannot stop a newer camera stream', { timeout: 2000 }, async () => {
  const h = harness({ native: false, bundled: true }); await h.start();
  h.stop(); await h.start(); const current = h.nodes['camera-video'].srcObject; assert.ok(current);
  await h.runTimers(); assert.equal(h.nodes['camera-video'].srcObject, current);
  assert.equal(current.stops, 0);
});
