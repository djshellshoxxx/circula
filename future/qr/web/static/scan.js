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
  var timer = 0;
  var running = false;
  var generation = 0;

  function say(msg) { status.textContent = msg; }
  function current(request) { return request === generation && !document.hidden; }

  function deliver(text, request) {
    if (!current(request)) return;
    stop();
    if (typeof text !== "string" || text.length === 0 || text.length > MAX) {
      say("That code could not be used. Enter it by hand instead.");
      return;
    }
    input.value = text;
    say("Code read. Review the next screen; nothing has been paid.");
    form.requestSubmit ? form.requestSubmit() : form.submit();
  }

  function releaseVideo(ownedVideo) {
    var media = ownedVideo.srcObject;
    if (media && typeof media.getTracks === "function") {
      media.getTracks().forEach(function (track) { track.stop(); });
    }
    ownedVideo.srcObject = null;
  }

  function stop() {
    generation++;
    running = false;
    if (timer) { clearTimeout(timer); timer = 0; }
    // Nimiq normally delays stream cleanup. Release the attached media before
    // clearing its reference, including streams that the library acquired.
    releaseVideo(video);
    if (scanner) { try { scanner.stop(); scanner.destroy(); } catch (e) {} scanner = null; }
    video.hidden = true;
    stopBtn.hidden = true;
    startBtn.hidden = false;
  }

  async function nativeScan(request, ownedVideo) {
    var det = new BarcodeDetector({ formats: ["qr_code"] });
    var acquired = await navigator.mediaDevices.getUserMedia({ video: { facingMode: "environment" }, audio: false });
    if (!current(request)) {
      acquired.getTracks().forEach(function (track) { track.stop(); });
      if (request === generation) stop();
      return;
    }
    ownedVideo.srcObject = acquired;
    await ownedVideo.play();
    if (!current(request)) {
      releaseVideo(ownedVideo);
      if (request === generation) stop();
      return;
    }
    running = true;
    (function tick() {
      if (!running || !current(request)) return;
      det.detect(ownedVideo).then(function (codes) {
        if (!running || !current(request)) return;
        if (codes.length) { deliver(codes[0].rawValue, request); } else { timer = setTimeout(tick, 200); }
      }).catch(function () { if (running && current(request)) timer = setTimeout(tick, 400); });
    })();
  }

  async function libScan(request, ownedVideo) {
    var mod = await import("/qr/static/vendor/qr-scanner/qr-scanner.min.js");
    if (!current(request)) {
      if (request === generation) stop();
      return;
    }
    var QrScanner = mod.default;
    var ownedScanner = new QrScanner(ownedVideo, function (r) { deliver(r && r.data, request); },
      { returnDetailedScanResult: true, preferredCamera: "environment", maxScansPerSecond: 8 });
    scanner = ownedScanner;
    await ownedScanner.start();
    if (!current(request)) {
      releaseVideo(ownedVideo);
      if (request === generation) stop();
      return;
    }
    running = true;
  }

  startBtn.addEventListener("click", async function () {
    if (!window.isSecureContext || !navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      say("The camera needs a secure (https) page and a supported browser. Enter the code by hand instead.");
      return;
    }
    stop();
    var request = generation;
    // A destroyed Nimiq instance can still finish delayed cleanup against its
    // video element. Give each attempt its own element so it cannot stop the
    // next attempt's media. Attributes and the visible element ID are retained.
    var ownedVideo = video.cloneNode(true);
    video.replaceWith(ownedVideo);
    video = ownedVideo;
    startBtn.hidden = true;
    stopBtn.hidden = false;
    video.hidden = false;
    say("Starting camera. Center the QR code in the picture.");
    try {
      if ("BarcodeDetector" in window) { await nativeScan(request, ownedVideo); } else { await libScan(request, ownedVideo); }
      if (current(request) && running) say("Camera on. Center the QR code in the picture.");
    } catch (e) {
      releaseVideo(ownedVideo);
      if (request !== generation) return;
      stop();
      say("The camera is not available or permission was denied. Enter the code by hand instead.");
    }
  });
  stopBtn.addEventListener("click", function () { stop(); say("Camera stopped."); });
  document.addEventListener("visibilitychange", function () { if (document.hidden) { stop(); say("Camera stopped because the page was hidden."); } });
  window.addEventListener("pagehide", stop);
  window.addEventListener("circula:logout", stop);
  form.addEventListener("submit", stop);

  fileIn.addEventListener("change", async function () {
    var f = fileIn.files && fileIn.files[0];
    // Any new selection supersedes pending decoding, even if it is rejected.
    if (f) stop();
    if (!f || f.size > 10 * 1024 * 1024) { say("Choose an image under 10 MB."); return; }
    var request = generation;
    // Retain the selected File locally; a cancelled decoder must not later
    // clear or submit a newer selection.
    fileIn.value = "";
    try {
      var mod = await import("/qr/static/vendor/qr-scanner/qr-scanner.min.js");
      if (!current(request)) return;
      var r = await mod.default.scanImage(f, { returnDetailedScanResult: true });
      deliver(r && r.data, request);
    } catch (e) {
      if (current(request)) say("No QR code was found in that photo. Try again or enter the code by hand.");
    }
  });
})();
