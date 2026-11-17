// Test helper: decode QR PNG pixel data with the bundled jsQR (Apache-2.0).
// Input on stdin: JSON array of {w,h,rgba(base64)}. Output: JSON array of decoded strings or null.
import jsQR from "./jsqr-es6/jsQR.js";
let s = "";
for await (const c of process.stdin) s += c;
const out = JSON.parse(s).map(({ w, h, rgba }) => {
  const d = new Uint8ClampedArray(Buffer.from(rgba, "base64"));
  const r = jsQR(d, w, h, { inversionAttempts: "dontInvert" });
  return r ? r.data : null;
});
process.stdout.write(JSON.stringify(out));
