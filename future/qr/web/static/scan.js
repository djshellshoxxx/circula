// Circula QR camera enhancement (progressive; the HTML form works without it).
// Original code. Uses the platform BarcodeDetector when present and otherwise
// the bundled Nimiq qr-scanner (MIT; see THIRD-PARTY-NOTICES.md), loaded only
// from this origin. Camera frames and decoded text are never stored, logged or
// sent anywhere except the visible form submit to the same-site /qr/manual.
(function () {
  "use strict";
  var section = document.getElementById("camera-section");
  var form = document.getElementById("manual-form");
  var input = document.getElementById("code");
  var video = document.getElementById("camera-video");
  var startBtn = document.getElementById("camera-start");
  var stopBtn = document.getElementById("camera-stop");
  var fileIn = document.getElementById("camera-file");
  var status = document.getElementById("camera-status");
  if (!section || !form || !input || !video || !startBtn || !stopBtn || !fileIn || !status) return;
  section.hidden = false;

  var MAX = 2048;
  var scanner = null;
  var stream = null;
  var timer = 0;
  var running = false;

  function say(msg) { status.textContent = msg; }

  function deliver(text) {
    stop();
    if (typeof text !== "string" || text.length === 0 || text.length > MAX) {
      say("That code could not be used. Enter it by hand instead.");
      return;
    }
    input.value = text;
    say("Code read. Review the next screen; nothing has been paid.");
    form.requestSubmit ? form.requestSubmit() : form.submit();
  }

  function stop() {
    running = false;
    if (timer) { clearTimeout(timer); timer = 0; }
    if (scanner) { try { scanner.stop(); scanner.destroy(); } catch (e) {} scanner = null; }
    if (stream) { stream.getTracks().forEach(function (t) { t.stop(); }); stream = null; }
    video.srcObject = null;
    video.hidden = true;
    stopBtn.hidden = true;
    startBtn.hidden = false;
  }

  async function nativeScan() {
    var det = new BarcodeDetector({ formats: ["qr_code"] });
    stream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: "environment" }, audio: false });
    video.srcObject = stream;
    await video.play();
    running = true;
    (function tick() {
      if (!running) return;
      det.detect(video).then(function (codes) {
        if (codes.length) { deliver(codes[0].rawValue); } else { timer = setTimeout(tick, 200); }
      }).catch(function () { timer = setTimeout(tick, 400); });
    })();
  }

  async function libScan() {
    var mod = await import("/qr/static/vendor/qr-scanner/qr-scanner.min.js");
    var QrScanner = mod.default;
    scanner = new QrScanner(video, function (r) { deliver(r && r.data); },
      { returnDetailedScanResult: true, preferredCamera: "environment", highlightScanRegion: true, maxScansPerSecond: 8 });
    await scanner.start();
    running = true;
  }

  startBtn.addEventListener("click", async function () {
    if (!window.isSecureContext || !navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      say("The camera needs a secure (https) page and a supported browser. Enter the code by hand instead.");
      return;
    }
    startBtn.hidden = true;
    stopBtn.hidden = false;
    video.hidden = false;
    say("Starting camera. Point it at the QR code.");
    try {
      if ("BarcodeDetector" in window) { await nativeScan(); } else { await libScan(); }
      say("Camera on. Point it at the QR code.");
    } catch (e) {
      stop();
      say("The camera is not available or permission was denied. Enter the code by hand instead.");
    }
  });
  stopBtn.addEventListener("click", function () { stop(); say("Camera stopped."); });
  document.addEventListener("visibilitychange", function () { if (document.hidden && running) { stop(); say("Camera stopped because the page was hidden."); } });
  window.addEventListener("pagehide", stop);

  fileIn.addEventListener("change", async function () {
    var f = fileIn.files && fileIn.files[0];
    if (!f || f.size > 10 * 1024 * 1024) { say("Choose an image under 10 MB."); return; }
    try {
      var mod = await import("/qr/static/vendor/qr-scanner/qr-scanner.min.js");
      var r = await mod.default.scanImage(f, { returnDetailedScanResult: true });
      deliver(r && r.data);
    } catch (e) {
      say("No QR code was found in that photo. Try again or enter the code by hand.");
    } finally {
      fileIn.value = "";
    }
  });
})();
