import {createRequire} from 'node:module';import assert from 'node:assert/strict';
const {chromium}=createRequire(import.meta.url)('playwright');
const origin=process.env.PWA_TEST_ORIGIN||'http://127.0.0.1:18761';
const browser=await chromium.launch({executablePath:process.env.CHROMIUM||'/usr/bin/chromium',headless:true,args:['--no-sandbox']});
try{
const context=await browser.newContext({viewport:{width:320,height:720}});const page=await context.newPage();const errors=[];page.on('pageerror',e=>errors.push(e.message));
await page.goto(origin+'/app/install');await page.evaluate(()=>navigator.serviceWorker.ready);await page.reload();
const cached=await page.evaluate(async()=>{const out=[];for(const n of await caches.keys()){const c=await caches.open(n);out.push({name:n,urls:(await c.keys()).map(r=>r.url)});}return out;});
assert.equal(cached.length,1);assert.equal(cached[0].urls.length,4);assert(cached[0].urls.every(u=>/\/app\/static\/[^/?]+\.[0-9a-f]{16}\.(css|js|png)$/.test(u)));
// A new worker activation removes only obsolete caches in this module namespace.
const obsolete=cached[0].name.replace(/-[^-]+$/, '-obsolete');
await page.evaluate(async name=>{await caches.open(name);await caches.open('unrelated-public-cache');const r=await navigator.serviceWorker.getRegistration();await r.unregister();},obsolete);
await page.reload();await page.evaluate(()=>navigator.serviceWorker.ready);await page.reload();
const names=await page.evaluate(()=>caches.keys());assert(!names.includes(obsolete));assert(names.includes('unrelated-public-cache'));

assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true);
const manifest=await (await context.request.get(origin+'/app/manifest.webmanifest')).json();assert.equal(manifest.scope,'/app/');assert.equal(manifest.icons.length,2);
await page.goto(origin+'/app/');await page.selectOption('select','A');await page.getByRole('button',{name:'Sign in',exact:true}).click();await page.getByText('Private synthetic balance for A: 100').waitFor();
await page.evaluate(async()=>{await fetch('/app/balance');await fetch('/app/inbox?token=secret');});
assert.equal(await page.evaluate(async()=>{let n=0;for(const k of await caches.keys())n+=(await(await caches.open(k)).keys()).length;return n}),4);
const before=await (await context.request.get(origin+'/app/counts')).text();
await context.setOffline(true);
const failed=await page.evaluate(async()=>{try{await fetch('/app/trade',{method:'POST',body:'intent=x'});return false}catch{return true}});assert.equal(failed,true);
await page.reload();await page.getByRole('heading',{name:'Circula is unavailable'}).waitFor();assert(!await page.locator('body').textContent().then(s=>s.includes('balance for A')));
await context.setOffline(false);await page.goto(origin+'/app/');assert.equal(await(await context.request.get(origin+'/app/counts')).text(),before);
await page.getByRole('button',{name:'Sign out',exact:true}).click();await page.getByRole('heading',{name:'Install Circula'}).waitFor();
await page.goBack();assert(!await page.locator('body').textContent().then(s=>s.includes('balance for A')));
await page.goto(origin+'/app/');await page.selectOption('select','B');await page.getByRole('button',{name:'Sign in',exact:true}).click();await page.getByText('Private synthetic balance for B: 100').waitFor();
// Synthetic session-end event is adapter contract; verify synchronous private DOM scrub.
await page.evaluate(()=>window.dispatchEvent(new Event('circula:session-ended')));assert.equal(await page.locator('[data-private]').textContent(),'');
await page.goto(origin+'/app/');await page.evaluate(()=>window.dispatchEvent(new PageTransitionEvent('pagehide',{persisted:true})));assert.equal(await page.locator('[data-private]').textContent(),'');
assert.deepEqual(errors,[]);await context.close();
const plain=await browser.newContext({javaScriptEnabled:false,viewport:{width:320,height:720}});const p=await plain.newPage();await p.goto(origin+'/app/');await p.selectOption('select','A');await p.getByRole('button',{name:'Sign in',exact:true}).click();await p.getByText('Private synthetic balance for A: 100').waitFor();await p.getByRole('button',{name:'Send synthetic trade'}).click();await p.getByText('Private synthetic balance for A: 100').waitFor();await p.getByRole('button',{name:'Sign out',exact:true}).click();await p.getByRole('heading',{name:'Install Circula'}).waitFor();await plain.close();
console.log('PASS: real Chromium install/CacheStorage, privacy, offline POST/no replay, account switch/back, DOM scrub, 320px and no-JS flows');
}finally{await browser.close();}
