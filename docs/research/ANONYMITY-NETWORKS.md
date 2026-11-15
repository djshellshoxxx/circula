# Anonymity networks for Circula: Tor onion services, I2P, Lokinet, Yggdrasil

Date 2026-10-21, base `claude/circula-finish` (1c248d9). Research only: no code changed.

## How to read the evidence tags

- **[V-search]** read in a search-result summary of a primary source (Tor man page, Tor spec, Tor Browser tickets, I2P/i2pd docs, Mozilla bugzilla). The sandbox proxy blocked `community.torproject.org`, `spec.torproject.org`, `gitlab.torproject.org`, `docs.i2pd.website`, `i2p.net`, `manpages.debian.org`, so no primary page was read in full.
- **[V-local]** checked here by running it or reading this repository.
- **[U]** from memory or inference, not verified. Verify before relying on it.
- No `tor`, `i2pd` or I2P router was installed or run. **No torrc or tunnels.conf below was executed.** The one thing run: the x25519 client-auth key recipe (OpenSSL 3.0.13), see section 1.4.

## 0. Recommendation (answer first)

1. Use **Tor v3 onion service** as the primary anonymous channel. Run it as a separate `tor` process (or container) that forwards to Circula on loopback. Offer two services:
   - **private** (v3 client authorization) for members, and
   - optionally **public** (no client auth) with PoW defence on, for sign-up/landing.
2. Do **not** use `HiddenServiceNonAnonymousMode`/`SingleHopMode` unless the goal is speed with no server anonymity. It is for services that are not hiding their location. [V-search]
3. Plain HTTP on `.onion` is acceptable (the onion layer is end-to-end encrypted and authenticated) **provided** Circula's `cookie-secure` stays `true` and clients are Tor Browser. Tor Browser treats `.onion` as a secure context, so `Secure` and `__Host-` cookies work there. [V-search for the secure-context part; `__Host-` over http `.onion` **[U]: test in a current Tor Browser before release**]
4. **Circula cannot serve onion users safely today without code changes**: every onion client appears as `127.0.0.1`, so per-IP limits and the per-(login, IP) lock collapse into one shared bucket (section 3). Minimum backlog: items B1, B2, B3, B4 (about 2 to 3 days).
5. **I2P is a viable second channel**, not the first. It is less used by end users, has no browser equivalent of Tor Browser, but gives a stable per-client key (destination hash) through `X-I2P-Dest*` headers on the Java I2P HTTP server tunnel [V-search for Java I2P; i2pd **[U]**]. Recommend only if the community already uses I2P.
6. Lokinet and Yggdrasil: not recommended (section 6).

## 1. Hosting Circula as an onion service

### 1.1 Topology

```
Tor Browser --(3 hops + 3 hops)--> tor daemon (onion service) --TCP--> 127.0.0.1:8080 circula
```

- Circula `listen` is TCP only (`net.Listen("tcp", ...)` in `cmd/circula/serve.go`) **[V-local]**, so use `HiddenServicePort 80 127.0.0.1:8080`. Tor also accepts a `unix:` target **[U]**; Circula has no unix-socket listen, so not usable without code (B9, optional).
- Bind Circula to loopback, or to a private Docker network that only the tor container joins. Never expose the same Circula listener on a clearnet interface if the onion is meant to be the only way in.
- Run the **public** and **private** services from the same Circula? One Circula, two `HiddenServiceDir`s pointing at two ports, so that the private service's access control is not bypassable: bind Circula on two listeners is not supported today **[V-local]** (single `listen`). Options: (a) one onion service with client auth for everyone (simplest, recommended); (b) two Circula instances (two databases, wrong); (c) one Circula behind a tiny reverse proxy that listens on two ports and sets a header naming the entry (needs a `trusted-proxies` entry and code to read it: B10). Recommend (a).

### 1.2 Tor configuration

Option names below were confirmed to exist in the Tor manual via search summaries **[V-search]**; they were not run **[U-run]**.

```
# /etc/tor/torrc  (tor >= 0.4.8 for PoW)  -- UNVERIFIED by execution
HiddenServiceDir /var/lib/tor/circula/
HiddenServiceVersion 3
HiddenServicePort 80 127.0.0.1:8080

# Intro-point rate limit (defaults 25/s rate; 0 = infinite, so do not set 0)
HiddenServiceEnableIntroDoSDefense 1
HiddenServiceEnableIntroDoSRatePerSec 25
HiddenServiceEnableIntroDoSBurstPerSec 200

# Proof-of-work puzzle (0.4.8.4+ per Tor forum; default 0). Man-page defaults: queue rate 250, burst 2500.
# Lower rate/burst = higher puzzle effort for clients. Size to what SQLite single writer can absorb.
HiddenServicePoWDefensesEnabled 1
HiddenServicePoWQueueRate 25
HiddenServicePoWQueueBurst 100

# Optional: the PROXY v1 header carrying a per-circuit id (see section 3)
HiddenServiceExportCircuitID haproxy

# Optional hardening against abusive streams
HiddenServiceMaxStreams 50
HiddenServiceMaxStreamsCloseCircuit 1
```

Notes:
- `HiddenServiceEnableIntroDoSDefense` rate 0 means infinite and effectively disables the defence. [V-search]
- PoW first shipped in tor 0.4.8.4 stable. [V-search, forum]
- Vanguards: tor >= 0.4.7 has "vanguards-lite" built in (4 layer-2 guards, rotation about 1 week). For a long-lived service, the Tor release notes still recommend the separate **vanguards** addon. [V-search] Caveats: reports that the addon breaks with conflux (`ConfluxEnabled 0` workaround) and that Whonix disabled it and Debian Trixie dropped the package. [V-search, secondary sources] Position: rely on built-in vanguards-lite; run the addon only if you can pin a version known to work with your tor.
- **HiddenServiceSingleHopMode / NonAnonymousMode**: experimental; one-hop circuits to the client rendezvous; the service is not location-anonymous; hidden service dir can never go back to anonymous mode; no SocksPort on the same instance; cannot run alongside a normal onion service in the same instance. [V-search] Do not use for Circula.
- OnionBalance: only needed for multi-instance availability. Circula is SQLite single-writer, so one instance only; skip. **[U-scope]**

### 1.3 Client authorization (private service)

Service side: `<HiddenServiceDir>/authorized_clients/<any-name>.auth`, one file per client, content `descriptor:x25519:<52-char base32 public key>`; malformed files are ignored; client auth is enabled only if at least one valid file loads; removing a file revokes access but only after tor restart (one Tor guide says reload; check your version). [V-search, Tor guide and rend-spec Appendix G]

Client side: set `ClientOnionAuthDir <dir>` in torrc (Tor Browser reads it from its own profile, and also offers a key prompt); file `<name>.auth_private` containing `<onion address without .onion>:descriptor:x25519:<52-char base32 private key>`. [V-search]

Key generation recipe, **[V-local] run with OpenSSL 3.0.13, output length 52 chars as required**:

```sh
openssl genpkey -algorithm x25519 -out k.pem
PRIV=$(openssl pkey -in k.pem -outform DER | tail -c 32 | base32 | tr -d '=')
PUB=$(openssl pkey -in k.pem -pubout -outform DER | tail -c 32 | base32 | tr -d '=')
echo "descriptor:x25519:$PUB" > authorized_clients/alice.auth        # service side
echo "<onion-without-.onion>:descriptor:x25519:$PRIV"               # client side .auth_private
shred -u k.pem
```

(`tail -c 32` of the DER is the raw key for both PKCS8 private and SPKI public x25519 keys. The base32 alphabet must be accepted by tor as uppercase RFC 4648 without padding; **[U]** whether Tor wants uppercase: the spec examples are uppercase.)

Properties and limits:
- Client auth hides the **descriptor**: nobody without a key can even connect or learn the intro points. It is the strongest DoS and enumeration defence available and removes the need for public rate-limiting on the login page.
- It is per-device, not per-person. Revocation needs a tor restart/reload and key rotation for a leaked key. Operationally this means Circula admins cannot onboard through the app alone; they must hand over the key out of band (state this in the runbook).
- Client auth makes members un-anonymous to the operator only as far as the key name: use random file names, and one key per member is a stable per-member identifier at the tor layer but it is **not** visible to Circula over TCP. **[U]** Tor does not tell the backend which key was used.

### 1.4 Containers, systemd, keys

- Directory: `HiddenServiceDir` must be owned by the tor user, mode 0700 (tor refuses looser). [U]
- Docker: two containers (tor, circula) on an internal network; tor torrc target `HiddenServicePort 80 circula:8080` (hostname in the target is accepted, **[U]**, else use a fixed IP); volume for `/var/lib/tor/circula` (named volume, not baked into image). Circula `listen 0.0.0.0:8080` inside the internal network only, with no published port. Circula's `checkTransport` refuses `http://` public URL with non-loopback `listen` unless `allow-insecure-http=true` **[V-local]**; inside a private container network that flag is needed, and its warning is about plaintext on a network that here is a private bridge, which is acceptable. Prefer `network_mode: "service:tor"` (shared netns) so Circula can keep `listen 127.0.0.1:8080` and no flag is needed **[U]**.
- systemd: a drop-in is enough: `tor@default` plus `circula.service` with `After=tor.service`. Circula is independent of tor at runtime.
- **Key backup**: `hs_ed25519_secret_key`, `hs_ed25519_public_key`, `hostname` in `HiddenServiceDir`. Loss = the onion address is gone forever; theft = permanent impersonation. Back up offline and encrypted, separate from Circula's backups (Circula's `backup-key` does not cover it). The `authorized_clients/` dir must be backed up too (or the key list recorded). Do not put these in the Circula backup archive. [U-ops]
- Onion address is derived from the key: a new key means a new URL and all `public-base-url` links, invitations and bookmarks break.

### 1.5 Onion-Location and clearnet pairing

`Onion-Location: http://<addr>.onion/<path>` is honoured by Tor Browser only on a page served over HTTPS and only when the value is an `http`/`https` URL with a `.onion` host; the header on a `.onion` page itself is not meaningful. [V-search] So it only helps if Circula **also** has a clearnet HTTPS deployment. It would leak the existence of the onion to anyone who loads the clearnet site. For a private onion do not advertise it. If wanted, set it in the TLS reverse proxy (`add_header Onion-Location ...`), not in Circula. No Circula change required.

### 1.6 HTTPS on the .onion, `public-base-url`, cookies

- `.onion` is end-to-end encrypted by the protocol; Let's Encrypt cannot issue for it; a CA (e.g. HARICA) can issue DV for .onion **[U]**; not worth it for a self-hosted app.
- Tor Browser marks `.onion` as potentially trustworthy (pref `dom.securecontext.whitelist_onions`), shows no insecure-password warning, and **Secure cookies set over http `.onion` are accepted**: Mozilla bug 1633015 fixed in Firefox 77, and the rest was upstreamed under 1618113. [V-search]
- Consequence A (good): Circula defaults `cookie-secure=true` (names `__Host-circula_session`, `__Host-cm_csrf`) **[V-local]**; keep it. `__Host-` requires `Secure`, `Path=/`, no `Domain`. Whether Tor Browser sets a `__Host-` cookie from an `http://x.onion` response is **[U]**: the search found nothing on `__Host-` and onion. **Test with current Tor Browser stable and Tor Browser Android before declaring support.** Fallback: `cookie-secure=false` gives plain names (loses SA2-14 sibling-subdomain protection, which is moot on a single onion host; no sibling subdomains exist).
- Consequence B (watch): Tor Browser sends a Secure cookie over http to the backend; the onion hop is encrypted, tor-to-Circula hop is local plaintext; that is fine on loopback. If an intermediate reverse proxy sits between tor and Circula on another host, that segment is plaintext. [V-search, Onionspray advisory]
- `public-base-url` must be `http://<addr>.onion`. `checkTransport` allows `http://` only for loopback hosts unless `allow-insecure-http=true` **[V-local]**: so today an onion deployment must set `allow-insecure-http=true`. That flag then also silences warnings for genuine misconfigurations. Backlog B1: accept `*.onion` (and `*.b32.i2p`) hosts as secure-transport without the flag, but still require loopback/private `listen`.
- `Edge.Secure(r)` returns true only for `ForceSecure`, TLS, or trusted proxy + `X-Forwarded-Proto: https`; with `cookie-secure=true` it is `ForceSecure`, so HSTS is emitted only if `PublicHTTPS` (https base URL) or `Secure(r)` is true **[V-local]**: on `http://x.onion` with `cookie-secure=true`, `Edge.HSTS` sends `Strict-Transport-Security` over a http onion response (since `ForceSecure` makes `Secure(r)` true). Browsers ignore HSTS on non-HTTPS responses; harmless, but B1 should skip it for onion base URLs. **[V-local reading, not run]**
- CSRF same-origin check compares `Origin` host to `public-base-url` host (`sameOrigin`) **[V-local]**. Tor Browser sends the onion host; fine when `public-base-url` equals the address the user typed. A user reaching the same onion via two names (e.g. through a gateway) fails CSRF: expected.

## 2. Clients over Tor

| Client | How | Notes |
|---|---|---|
| Tor Browser (desktop, Android) | open the .onion URL; for client auth, Tor Browser prompts for the key and stores it in its profile | Best path. JS disabled at "Safest" works: Circula's CSP is `default-src 'none'; style-src 'self'; script-src 'self'` (no inline) **[V-local]**; check pages degrade without JS (not tested here) |
| Onion Browser / Brave Tor window / Orbot | partial | Brave private-with-Tor is weaker (shared circuits, no client-auth UI) **[U]** |
| curl | `curl --socks5-hostname 127.0.0.1:9050 http://x.onion/...` (`socks5h://` in `ALL_PROXY` also works) | `--socks5` (without hostname) resolves locally and leaks the lookup; never use. Tor's SOCKSPort must have `ClientOnionAuthDir` for private services **[U]** |
| `torsocks` | `torsocks curl ...` | blocks local DNS; needs tor SocksPort **[U]** |
| Go API client | `proxy.SOCKS5("tcp","127.0.0.1:9050",nil,proxy.Direct)` from `golang.org/x/net/proxy`; hand the dialer's `DialContext` to `http.Transport`. Go's SOCKS5 client sends the hostname to the proxy, no local DNS lookup. [V-search: go-ricochet code and x/net commit for `socks5h`] | Go's `net/http` also accepts `socks5h://` proxy URL (golang/go#24135) [V-search] |
| API tokens | `Authorization: Bearer ...` works unchanged over onion | No cookies, so none of the `__Host-` questions apply. Tokens in URLs must never be used (Referer policy is `same-origin` **[V-local]**) |

Issues to document for users:
- Private service + `ClientOnionAuthDir`: tor needs a restart/reload when a new key is added (SIGHUP is enough on recent versions [U]).
- Latency: first request builds a rendezvous circuit (seconds). API clients need generous timeouts (30 to 60 s) and retries. Rate-limits keyed on time windows must not punish retries. [U]
- No certificate issues: `http://` onion has no cert. Do **not** use `https://` on the `.onion` with a self-signed cert (Tor Browser warns).
- Mixed content: Circula loads no external assets (CSP `default-src 'none'` allows only `'self'`) **[V-local]**, so no clearnet fetch can leak from an onion page.

## 3. Rate limiting and lockout when every client is 127.0.0.1

### 3.1 What breaks today (all **[V-local]**)

- Web: `internal/web/session.go` `clientIP` -> `Edge.RateKey` -> `ClientIP`. With no trusted proxy the key is the TCP peer, `127.0.0.1` for every onion client. `s.reg` (registration/setup limiter) is one bucket for the whole onion service.
- API: `internal/api/middleware.go:53` `rl.allow("ip:"+RateKey, IPPerMinute)`: one shared per-IP budget for all onion clients; one abusive client can starve everyone else. Per-principal limits (`PrincipalPerMinute`) still work.
- Login throttle (`internal/service/throttle.go`, SA2-10): key (login, IP) locks at 5 failures, global login key locks at 100 failures; max lock 15 min. With all clients on one IP, the (login, 127.0.0.1) row **is** effectively per-login: five bad guesses from anyone lock the victim for 15 min, which re-opens the SA2-10 lock-DoS on the onion (attacker needs only a Tor client, and no IP cost). The 100/login global cap is the only brake on guessing and is far weaker than the per-IP cap.
- `trusted-proxies` set to loopback does not help: tor sends no `X-Forwarded-For`.
- The IPv6 /64 keying (SA2-15) is irrelevant for onion peers.

### 3.2 What Tor gives you

- There is **no client IP and no stable client id** in a normal onion connection. Tor can export a **per-circuit id** with `HiddenServiceExportCircuitID haproxy`: tor then prefixes each TCP connection with a PROXY v1 line `PROXY TCP6 SRC DST SPORT DPORT\r\n`; the low 32 bits of SRC encode a unique circuit id (UCircId); the upper 96 bits and the ports must be ignored; DST should be `::1`. Only for v3 services, C tor only. [V-search, Tor spec "haproxy-proxy" and man page]
  - The brief assumed per-circuit was unavailable; it **is** available but needs Circula to parse the PROXY header on its listener (Go's `net/http` does not do this natively: wrap the `net.Listener`, B3).
  - A circuit id is **not an identity**: a client can build circuits freely (a new circuit per attempt), so a per-circuit limit adds a cheap concurrency/fairness bucket but does not stop a determined guesser. It does separate honest members from an attacker that holds one circuit open, and it lets Circula tell "all on one circuit" (one Tor client) from many. Moderate value. Also the id is unique per circuit while the circuit lives; reuse after close is possible **[U]**.
- **Client authorization** gives an access-control layer that makes login brute-force by outsiders impossible: only key holders reach the login page. For a private service this is the main answer. Residual risk: a key holder (insider) can lock another member. Limit blast radius with (login, circuit-id) pair keys and operator unlock (exists: `circula unlock`).
- **PoW** (HiddenServicePoWDefensesEnabled) raises the cost of connection flooding at the rendezvous; it does not rate-limit logins but bounds the request rate that can reach Circula. [V-search]
- I2P `X-I2P-DestHash` is a stable per-client key (section 5).

### 3.3 Recommended design (no code written)

1. **Mode switch**: new setting `client-key-source` (`remote-addr` default | `proxy-v1-circuit` | `header:<name>`), and a derived boolean `anonymous-transport` set automatically when `public-base-url` host ends in `.onion` or `.i2p`.
2. When `anonymous-transport`:
   - Per-IP limiters (web `s.reg`, API `IPPerMinute`) switch from per-IP to **global caps per route class** (login attempts per minute for the whole service, registration per hour, setup attempts) plus optional per-circuit-id buckets when available. Numbers: login global 30/min, burst 60 **[U, tune]**.
   - Login lock: keep the two rows but make the **pair row** (login, circuit-id or DestHash) the lock for the attacker side; raise the **per-login global** threshold from 100 to a documented value and add **exponential back-off delay (not lock)**: each failure for a login adds a delay (e.g. 1, 2, 4 ... capped 5 s) applied to the response of *all* attempts for that login, so a lock never denies the true owner outright. The correct owner who already holds a session or client-auth key is not blocked.
   - **Correct-password-bypasses-pair-lock is unsafe**, don't. Instead let a valid TOTP/recovery path or operator unlock clear the lock.
   - Prefer **per-login global cap with owner exemption by a second factor already satisfied** only if MFA is on; **[U]** design decision for owner.
3. Optional **captcha-free token** gate for the public service (sign-up): a one-time invitation code or a per-network "join secret" in the URL (Circula already has invitations and `CIRCULA_SETUP_SECRET`) **[V-local]**, plus Tor's PoW. No image captcha: it would need JS or external assets and conflicts with CSP and privacy.
4. Operator visibility: `/metrics` counters `login_throttled_total{scope}` split by pair/global/circuit, because under onion a spike is a DoS signal, not a typo.
5. Document that with client auth **off** and a public service, an attacker can lock any account on the onion for the lock duration (15 min) by sending 5 bad logins; the mitigations above make that a delay, not a lock.

## 4. Logging, metadata, privacy

Threat model for the onion deployment: (a) network observer: sees only Tor traffic to the operator's tor; (b) onion service operator: sees who uses the service through Circula's own data (accounts, trades), not their IP; (c) co-tenant/host compromise: reads database, tor keys; (d) adversary correlating timing: login times, trade events, backup times vs onion availability; (e) deanonymizing the server: the server's real IP must not leak via Circula (links, mail, SMTP, DNS lookups, update checks).

What Circula must not leak (check list, with state from code):
- **public-base-url in absolute links**: `h_admin.go:296` builds invitation links from `public-base-url` **[V-local]**. For an onion deployment it must be the onion URL only; never mix the clearnet URL. Mail: Circula has no SMTP code in `internal`/`cmd` (grep found none) **[V-local]**, so there is no email path to leak the server IP. Note in the doc that if a notifier is ever added, it must not send mail from the onion host directly (reveals IP).
- **Host header**: Circula accepts any Host when `public-base-url` is empty and compares Host in `sameOrigin`. An onion service receives `Host: <addr>.onion` as typed by the client. With `public-base-url` set, the CSRF origin check uses the configured host. No Host-header allow-list exists; add (B6) so that a request with an unexpected Host (including the clearnet IP) is refused: prevents a cache or link poisoning and "who else points at this server".
- **Logs**: the startup log prints `public_base_url` and `listen` **[V-local]**, fine. Anything logging `RemoteAddr`: in onion mode it is 127.0.0.1 (nothing to leak). Check access logs in front proxies (nginx, haproxy) do not store circuit ids longer than needed (circuit id is pseudonymous). Set journald retention short. **[U, not audited line by line]**
- **Metadata Circula itself stores**: `X-Request-Id`, audit log with timestamps, `created_at` of trades, session lifetimes: all coarse timing that is correlatable with an observer of the tor daemon. Mitigation: timestamps at coarse granularity are a product decision; not recommended to change.
- **No external assets, no analytics**: CSP is `default-src 'none'; ... img-src 'self'`, `Referrer-Policy: same-origin`, `Cache-Control: no-store` **[V-local]**. Good for onion. `internal/web/templates/base.html` references only `/static/style.css` and `/static/app.js` **[V-local, base.html only; other templates not individually read]**. Keep a CI check (B11).
- **Backups**: backups contain the database (possibly with onion-visible account data) but not the tor keys; they are encrypted when `backup-key` set. Where backup copies go (clearnet object storage) reveals the host. Pair with an ops note: do not run scheduled upload from the same host that must stay hidden unless uploads also go over Tor. **[U]**
- **Time sync**: a skewed clock breaks onion descriptors (tor needs consensus-valid time) and leaks clock fingerprint; use the host's NTP, do not expose NTP info via headers. Circula sends `Date` (Go default) which gives host clock to clients; harmless but a fingerprint **[U]**.
- **Server fingerprint over time**: uptime and `/metrics`, `/healthz` must not be reachable via onion (metrics is already refused for proxy headers and non-loopback allow-list; tor connects from loopback, so `metrics-allow 127.0.0.1` **would allow onion users to read `/metrics`**: **[V-local reading]** `ops_http.go` internal check uses `RemoteAddr` and refuses requests carrying proxy headers; tor adds none. Set `CIRCULA_METRICS_TOKEN` (documented for reverse proxies on the same host) in every onion deployment. This is a **concrete finding**: B2.)

## 5. I2P

### 5.1 Routers

| | i2pd (C++) | Java I2P |
|---|---|---|
| Footprint | low CPU/RAM (anecdotal [V-search, forum, weak]) | heavier JVM, console |
| Config | `i2pd.conf`, `tunnels.conf` (INI) | web console `i2ptunnel` (or `i2ptunnel.config`) |
| SAM v3 | on by default; lacks AUTH, PING/PONG, SESSION ADD/REMOVE, most 3.2/3.3 extensions (as of i2pd 2.58.0) [V-search, i2p.net SAMv3 table] | enable in `/configclients`; fuller feature set |
| Tunnel quantity default | 5 | 2 [V-search] |
| Transports | SSU2 default, SSU off since 2.44.0 [V-search] | NTCP2, SSU2 |
| Server-tunnel headers | HTTP server tunnel: `hostoverride` option for Host [V-search]; `X-I2P-Dest*` injection **[U]** (not confirmed) | HTTP server tunnel adds `X-I2P-DestHash`, `X-I2P-DestB64`, `X-I2P-DestB32`; client cannot spoof them [V-search, I2P i2ptunnel doc] |
| Reachability | no primary-source text found; both need inbound reachability or introducers (SSU2). Router must show "reachable"/not firewalled for stable service **[U]** | same **[U]** |

### 5.2 i2pd server tunnel (**[U]** syntax, unverified; options individually seen in i2pd docs search results: `type`, `host`, `port`, `keys`, `inport`, `gzip`, `signaturetype`, `hostoverride`; keys file auto-created if missing [V-search])

```ini
# tunnels.conf
[circula]
type = http
host = 127.0.0.1
port = 8080
keys = circula.dat
# default signature type for server tunnels is 7 (EdDSA) in current i2pd; older docs said 1 [V-search conflicting; omit and take the default]
```

- Address: the b32 (`<52 chars>.b32.i2p`) is derived from `circula.dat`; show it in the i2pd web console or logs. A human name needs address-book registration (jump service/registrar) - skip for private use.
- Destination keys = secret: back up `circula.dat` like the tor keys.
- Clients: browser through i2pd/Java I2P **HTTP proxy** (default 127.0.0.1:4444 [U]); curl `--proxy http://127.0.0.1:4444`. I2P HTTP proxy replaces the Host header with a b32 address and strips `Accept-*`, `Referer`, `Via`, `From` [V-search, I2P docs]. No Tor-Browser equivalent: fingerprinting of the browser is the user's problem.
- App-level clients: **SAMv3** for embedding (stream sessions); Go libraries exist (e.g. `go-i2p/sam3`, `eyedeekay/sam3`) **[U]**; not needed for Circula as server.
- Encrypted LeaseSet with per-client keys exists in I2P ("encrypted leaseset", per-client auth) **[U]**, so I2P has an analogue of Tor client auth; not verified.

### 5.3 Circula-specific points for I2P

- Same `127.0.0.1` collapse. Java I2P's `X-I2P-DestHash/B32/B64` gives a **stable per-client key**: **good** for rate-limiting and locks (key = (login, DestHash)), but is a **tracking identifier** across logins and sessions for the operator; document the privacy trade-off. Treat the header as trusted **only** from the loopback peer (a new `client-key-source=header:X-I2P-DestHash` plus mandatory `trusted-proxies`-style peer allow-list), otherwise any HTTP client can forge it. With i2pd the header may be absent **[U]**: then fall back to global caps.
- Same Host/CSRF/`public-base-url=http://<b32>.b32.i2p` behaviour: browsers do **not** treat `.i2p` as a secure context (no equivalent of the Tor `.onion` exception) **[U]**. `Secure` cookies and `__Host-` would then be rejected over `http://x.b32.i2p` in Firefox/Chrome. So **`cookie-secure=true` breaks login over I2P with ordinary browsers**; needs `cookie-secure=false` (plain names) **[U, test]**. This is a significant difference vs Tor Browser.
- Latency/availability: tunnels rebuild every 10 min; first load of a b32 can take tens of seconds. [U]
- Anonymity set smaller and weaker documentation for service-hiding than Tor; no equivalent of Tor Browser's uniform fingerprint. Good for long-lived P2P and eepsite culture.

### 5.4 Compare

| Aspect | Tor v3 onion | I2P (eepsite) |
|---|---|---|
| Client software | Tor Browser (uniform fingerprint, secure-context for .onion) | any browser through router proxy; no uniform fingerprint |
| Access control | v3 client auth (descriptor encryption) [V-search] | encrypted leaseset per-client **[U]** |
| DoS defence | intro rate limit, PoW (0.4.8+), vanguards [V-search] | few; tunnel limits **[U]** |
| Per-client key at backend | none; optional per-circuit id via PROXY v1 [V-search] | `X-I2P-Dest*` (Java I2P; i2pd **[U]**) |
| Secure cookies | yes in Tor Browser [V-search] | no secure context for `.i2p` **[U]** |
| Hosting cost | tor daemon, tiny | router needs bandwidth share, reachability |
| Maturity for hidden web services | highest | ok |
| Support in Circula today | needs B1 to B4 | needs B1, B3-header, cookie decision |

## 6. Lokinet and Yggdrasil (brief)

- **Lokinet** (Oxen): SNApps are `.loki` services; a `keyfile` setting makes the address persistent; DHT publish flag, `mapaddr`; `lokinet -g` generates a config. [V-search, old 2020 posts and nixpkgs]. Needs a VPN-style TUN adapter on every client, small network and uncertain maintenance status in 2026 **[U]**; no browser integration, no client-auth. Not recommended.
- **Yggdrasil**: an encrypted IPv6 overlay mesh, **not anonymous** (peers see and the routing uses your public key/IPv6 address) **[U]**; web service reachable at `http://[200:...]`. Suitable for private reachability between trusted peers (like Tailscale/WireGuard), not for anonymity. Out of scope. No primary source read.

## 7. Matrix

| Criterion | Tor private (client auth) | Tor public | I2P (Java) | I2P (i2pd) | Lokinet | Yggdrasil |
|---|---|---|---|---|---|---|
| Server location hidden | yes | yes | yes | yes | yes (claimed) | no |
| Client anonymity | yes (Tor Browser) | yes | yes-ish | yes-ish | claimed | no |
| Outsiders can reach login | no | yes | yes | yes | yes | yes |
| Uniform client fingerprint | yes | yes | no | no | no | no |
| Secure cookies / `__Host-` | yes (test `__Host-`) | same | no (likely) | no (likely) | no | no |
| Per-client key for limits | circuit id (optional) | circuit id | DestHash header | maybe | ? | peer IPv6 |
| DoS controls | intro limit, PoW, auth | intro limit, PoW | weak | weak | weak | n/a |
| Circula changes needed | B1-B4 | B1-B5 | B1, B3, B7 | same + verify headers | not assessed | none |

## 8. Implementation backlog for Circula (not implemented)

Effort in ideal engineer-days incl. tests and docs. Owner decisions marked OD.

| ID | Item | Why | Effort |
|---|---|---|---|
| B1 | Treat `*.onion` (and `*.b32.i2p`) `public-base-url` as secure transport in `checkTransport` (no `allow-insecure-http` needed, still require loopback/private `listen` or explicit flag); skip HSTS for non-HTTPS base URL; startup log "anonymous transport mode" | today onion deployment forces a flag that disables the SEC-022 guard | 0.5 |
| B2 | Warn (or refuse) at start if anonymous-transport mode and `metrics-token` empty; make `ops_http.go` refuse `/metrics`, `/healthz` details when `anonymous-transport` unless token set | tor connects from loopback, so loopback allow-list exposes metrics to every onion user | 0.5 |
| B3 | Client-key source abstraction in `httpx.Edge`: `remote-addr` (default), `proxy-v1` (listener wrapper parsing PROXY v1 TCP6 line from loopback peer only; use low 32 bits as circuit id), `header:<Name>` (e.g. `X-I2P-DestHash`, accepted only from trusted peer). Feed `RateKey` and `WithClientIP` | per-IP controls are meaningless otherwise | 2 |
| B4 | Anonymous-mode throttle policy in `service/throttle.go` and web/API limiters: global per-route caps, per-login delay back-off instead of lock for global row, documented thresholds, metrics by scope; tests for SA2-10 lock-DoS on onion | restore SA2-10 guarantees | 2 (OD on policy) |
| B5 | Public-onion extras: sign-up gated by invitation or join secret only when `anonymous-transport`; no captcha | public service abuse | 1 |
| B6 | Host allow-list derived from `public-base-url` (refuse other Hosts, except loopback health checks) | no Host-based leakage, no cache or link poisoning | 0.5 |
| B7 | `cookie-secure` guidance/auto: when base URL host is `.i2p` default to false with startup note; keep `.onion` true; add a `Secure`-cookie browser test checklist | I2P browsers lack the secure-context exception | 0.5 |
| B8 | Docs: `docs/ops/onion-i2p.md` (this commit), link from `docs/ops/README.md` and `configuration.md` | | 0.25 |
| B9 | Optional: `listen unix:/path` support (and 0660 socket mode, group) for tor-to-Circula hop | tighter than loopback TCP on shared hosts | 1 |
| B10 | Optional: two listeners (public vs private entry) with different limits and policy | separate public and private onion on one DB | 2 |
| B11 | CI check: no absolute `http(s)://` to other origins in templates/static | no external assets guarantee | 0.25 |
| B12 | Manual test plan in `docs/qa`: Tor Browser stable/Android: login, `__Host-` cookie set, CSRF, no-JS pages, client auth key prompt | closes the [U] items | 1 |

Total for the Tor minimum (B1-B4, B6, B8, B12): about 7 days; I2P add-on B7 plus verification: about 1.5 days.

## 9. Unverified / open items

- Every torrc and tunnels.conf line: not executed (no tor/i2pd available); option names seen only via summaries of man pages and docs.
- `__Host-` cookie acceptance over `http://x.onion` in Tor Browser (including Android): not found in any source.
- Whether `unix:` targets and container hostname targets work in `HiddenServicePort`.
- i2pd injection of `X-I2P-Dest*` headers; PROXY protocol on i2pd server tunnels (none found).
- Secure-context status of `.i2p` in browsers.
- Revocation: restart vs reload for `authorized_clients` changes (sources disagree).
- Tor Browser Android client-auth UI details.
- Lokinet maintenance state in 2026; all Yggdrasil statements.
- Template grep for external URLs was shallow.

## 10. Sources

- Tor client authorization guide: https://community.torproject.org/onion-services/advanced/client-auth/ and rend-spec Appendix G: https://spec.torproject.org/rend-spec/client-authorization.html
- Tor onion service DoS guidelines: https://community.torproject.org/onion-services/advanced/dos/ ; PoW release: https://forum.torproject.org/t/proof-of-work-pow-defense-for-onion-service-is-released/8887 ; tor(1): https://manpages.debian.org/testing/tor/tor.1.en.html
- HAProxy/circuit id: https://spec.torproject.org/haproxy-proxy.html
- Vanguards-lite: tor 0.4.7.1-alpha release notes (blog.torproject.org); Whonix vanguards page https://www.whonix.org/wiki/vanguards (secondary)
- Single onion: https://bugs.torproject.org/20484 ; https://gitlab.torproject.org/tpo/core/tor/-/issues/40691
- Onion-Location: https://www.ctrl.blog/entry/tor-onion-location-header/ (secondary), Tor Browser spec proposal 100 (archives.seul.org)
- `.onion` secure context and cookies: https://bugzilla.mozilla.org/show_bug.cgi?id=1382359 , https://bugzilla.mozilla.org/show_bug.cgi?id=1633015 , Onionspray advisory 001 https://onionservices.torproject.org/apps/web/onionspray/security/advisories/001-torbrowser/
- Go SOCKS5: https://golang.org/issue/24135 ; x/net/proxy `socks5h` commit https://go.googlesource.com/net/+/395948e2f546cb82afa9e1f6d1a6e87849b9af1d
- i2pd tunnels: https://docs.i2pd.website/en/latest/user-guide/tunnels/ ; wiki https://github.com/purplei2p/i2pd/wiki/tunnels.conf
- I2PTunnel / headers: https://i2p.net/en/docs/api/i2ptunnel/ ; SAMv3: https://i2p.net/en/docs/api/samv3/
- Lokinet SNApps: https://loki.network/2020/06/17/lokinet-snapps-power-potential
