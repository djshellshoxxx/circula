import {build} from 'esbuild';import {readFile,writeFile,mkdir,readdir,rm} from 'node:fs/promises';import {createHash} from 'node:crypto';import {MAX_ASSET_BYTES} from '../src/precache.mjs';
await mkdir('public',{recursive:true});for(const f of await readdir('public'))await rm(`public/${f}`);
const assets=[];
async function asset(name,bytes,type){if(bytes.length>MAX_ASSET_BYTES)throw new Error(`asset ${name} exceeds ${MAX_ASSET_BYTES} bytes`);const h=createHash('sha256').update(bytes).digest();const file=name.replace(/(\.[^.]+)$/,`.${h.toString('hex').slice(0,16)}$1`);await writeFile('public/'+file,bytes);assets.push({file,sha256:h.toString('base64'),type,size:bytes.length});}
await asset('style.css',await readFile('src/style.css'),'text/css');
const client=await build({entryPoints:['src/client.mjs'],bundle:true,write:false,minify:true,format:'iife',target:'es2020'});await asset('client.js',client.outputFiles[0].contents,'application/javascript');
for(const n of [192,512])await asset(`icon-${n}.png`,await readFile(`src/icon-${n}.png`),'image/png');
const worker=await build({entryPoints:['src/worker.mjs'],bundle:true,write:false,minify:true,format:'iife',target:'es2020',define:{'process.env.NODE_ENV':'"production"'},legalComments:'inline'});await writeFile('public/worker.js',worker.outputFiles[0].contents);
await writeFile('public/catalog.json',JSON.stringify({version:createHash('sha256').update(JSON.stringify(assets)).update(worker.outputFiles[0].contents).digest('hex').slice(0,16),assets},null,2)+'\n');
