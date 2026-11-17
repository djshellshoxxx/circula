package web

// css is the only stylesheet. It is embedded inline and pinned in the page's
// Content-Security-Policy by hash, so no 'unsafe-inline' is needed. A host
// that has its own stylesheet can ignore it by rendering the view models
// itself; this keeps the module usable and accessible on its own.
const css = `
:root{color-scheme:light dark;--fg:#0b0c0c;--bg:#fff;--mut:#505a5f;--line:#b1b4b6;--link:#1d4ed8;--btn:#00703c;--btnfg:#fff;--warn:#b91c1c;--focus:#ffdd00}
@media (prefers-color-scheme:dark){:root{--fg:#f3f2f1;--bg:#0b0c0c;--mut:#b1b4b6;--line:#505a5f;--link:#93c5fd;--btn:#2fa86b;--btnfg:#0b0c0c;--warn:#fca5a5}}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--fg);font:1rem/1.5 system-ui,-apple-system,"Segoe UI",Roboto,sans-serif}
main{max-width:46rem;margin:0 auto;padding:1rem}
h1{font-size:1.75rem;margin:.5rem 0 1rem}h2{font-size:1.3rem;margin:1.5rem 0 .5rem}h3{font-size:1.1rem;margin:0}
a{color:var(--link)}a:focus-visible,button:focus-visible,input:focus-visible,textarea:focus-visible,summary:focus-visible{outline:3px solid var(--focus);outline-offset:2px;box-shadow:0 0 0 5px var(--fg)}
.skip{position:absolute;left:-999px}.skip:focus{left:1rem;top:1rem;background:var(--bg);padding:.5rem;z-index:2}
.vh{position:absolute;width:1px;height:1px;margin:-1px;padding:0;overflow:hidden;clip:rect(0 0 0 0);white-space:nowrap;border:0}
.hint{color:var(--mut);margin:.1rem 0 .4rem}
label{display:block;font-weight:600}
input[type=text],textarea{width:100%;min-height:2.75rem;padding:.5rem;border:2px solid var(--fg);border-radius:0;background:var(--bg);color:var(--fg);font:inherit}
.check{display:flex;gap:.6rem;align-items:center;min-height:2.75rem}.check input{width:1.5rem;height:1.5rem}.check label{font-weight:400}
button,.btn{min-height:2.75rem;min-width:2.75rem;padding:.4rem .9rem;border:2px solid var(--btn);background:var(--btn);color:var(--btnfg);font:inherit;font-weight:600;border-radius:4px;cursor:pointer;text-decoration:none;display:inline-block}
button.secondary{background:transparent;color:var(--fg);border-color:var(--fg)}
button.danger{background:transparent;color:var(--warn);border-color:var(--warn)}
ul.fav{list-style:none;padding:0;margin:0}
ul.fav>li{border-top:2px solid var(--line);padding:1rem 0}
.row{display:flex;flex-wrap:wrap;gap:.5rem;margin-top:.6rem}.row form{margin:0}
.badge{display:inline-block;border:2px solid var(--fg);padding:0 .4rem;margin-left:.4rem;font-size:.85rem;font-weight:700}
.err{border:4px solid var(--warn);padding:1rem;margin:0 0 1rem}.err h2{margin-top:0;color:var(--warn)}
.ok{border:4px solid var(--btn);padding:.75rem 1rem;margin:0 0 1rem}
.field-err{color:var(--warn);font-weight:700;margin:.1rem 0}
details{margin:0}summary{min-height:2.75rem;display:inline-flex;align-items:center;cursor:pointer;padding:0 .5rem;border:2px solid var(--fg);font-weight:600}
dl{margin:0}dt{font-weight:700;margin-top:.8rem}dd{margin:0}
nav.pager{margin:1rem 0}
@media (forced-colors:active){button,.btn,summary{border:2px solid ButtonText}}
`
