// Optional progressive enhancement. No storage of credentials, HTML or balances.
import {initPrivacy, registerWorker, validBase} from './privacy.mjs';
const config = document.querySelector('meta[name="circula-pwa"]');
const base = config && validBase(config.content) ? config.content : '/';
// Privacy lifecycle never depends on a valid base, service worker or channel.
initPrivacy({document, window, location, BroadcastChannel: globalThis.BroadcastChannel, base});
if (config) registerWorker({navigator}, config.content);
