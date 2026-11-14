# Running Circula as a Tor onion service or I2P eepsite

Status: guidance, **not yet supported by the code**. Read the research behind it, including the evidence tags and everything marked unverified, in [`docs/research/ANONYMITY-NETWORKS.md`](../research/ANONYMITY-NETWORKS.md). No snippet here was run against a real `tor` or `i2pd`.

## Before you start: known gaps in Circula today

1. Every onion or I2P client reaches Circula from `127.0.0.1`. Per-address rate limits are then one shared bucket, and the login lock (SA2-10) is effectively per login: five bad guesses from anyone lock a person for 15 minutes. Use Tor **client authorization** (private service) until the anonymous-transport throttle policy (backlog B3, B4) exists.
2. `public-base-url` of `http://<addr>.onion` is refused unless `allow-insecure-http=true` (SEC-022). Set it only when `listen` is loopback or a private container network that only the tor process joins.
3. `/metrics` is allowed from loopback, and tor connects from loopback without proxy headers. **Always set `CIRCULA_METRICS_TOKEN`** (or `metrics-token-file`), and `metrics-allow none` if you do not scrape.
4. Circula has no email sending, so no address leaks that way. Invitation links come from `public-base-url`: set it to the onion URL only.

## Tor: private onion service (recommended)

`/etc/tor/torrc` (tor 0.4.8 or newer for PoW; option names seen in the Tor manual, syntax **unverified by execution**):

```
HiddenServiceDir /var/lib/tor/circula/
HiddenServiceVersion 3
HiddenServicePort 80 127.0.0.1:8080
HiddenServiceEnableIntroDoSDefense 1
HiddenServicePoWDefensesEnabled 1
```

Do **not** set `HiddenServiceSingleHopMode` or `HiddenServiceNonAnonymousMode`: they remove the server's anonymity and cannot be undone for that directory.

Circula (`/etc/circula/circula.json` or the environment, see [configuration.md](configuration.md)):

```
listen                127.0.0.1:8080
public-base-url       http://<your-address>.onion
allow-insecure-http   true        # SEC-022 acknowledgement: the hop tor -> circula is local; the onion hop is encrypted
cookie-secure         true        # keep: Tor Browser accepts Secure cookies from .onion; test __Host- (research doc section 1.6)
trusted-proxies       none
CIRCULA_METRICS_TOKEN <random>
```

### Client authorization

Per member, on a trusted machine (the key-generation commands were run with OpenSSL 3.0.13 and produce 52-character base32 keys):

```sh
openssl genpkey -algorithm x25519 -out k.pem
PRIV=$(openssl pkey -in k.pem -outform DER | tail -c 32 | base32 | tr -d '=')
PUB=$(openssl pkey -in k.pem -pubout -outform DER | tail -c 32 | base32 | tr -d '=')
shred -u k.pem
# server: /var/lib/tor/circula/authorized_clients/<random-name>.auth
echo "descriptor:x25519:$PUB"
# member: <name>.auth_private in the directory named by ClientOnionAuthDir (Tor Browser: paste into its key prompt)
echo "<onion address without .onion>:descriptor:x25519:$PRIV"
```

Send the private line to the member over a channel you trust, once. Revoke by deleting the `.auth` file and reloading or restarting tor (sources differ on which; restart is certain). Rotate the key if a device is lost.

### Keys and backups

- Back up `hs_ed25519_secret_key`, `hs_ed25519_public_key`, `hostname` and `authorized_clients/` offline and encrypted, **separately** from Circula's backups. Losing the secret key loses the address for good; a stolen key lets someone impersonate your service.
- `HiddenServiceDir` must belong to the tor user, mode 0700.

### Docker or systemd

- Two units or containers: `tor` and `circula`. No clearnet port published for Circula.
- Docker: either a shared network namespace (`network_mode: "service:tor"`, so Circula keeps `listen 127.0.0.1:8080`) or an internal-only network with `HiddenServicePort 80 circula:8080` (hostname targets **unverified**). Keep the `HiddenServiceDir` on a named volume.
- systemd: `circula.service` independent of `tor.service`; no ordering needed beyond starting both.

### Public onion service (optional)

Without client authorization anyone can reach the login and sign-up pages. Keep PoW and the intro-point defence on, expect the lock-out issue in gap 1, and gate sign-up with invitations. Do not publish it before B3 and B4 exist.

### Onion-Location

Only useful if Circula also has a clearnet HTTPS site, and Tor Browser honours it only on HTTPS pages. It reveals that the onion exists. Set it in the TLS reverse proxy, not in Circula. Do not advertise a private onion.

## Using the service

- **Tor Browser** (desktop or Android): open `http://<addr>.onion/`, supply the client key when asked. Test sign-in, CSRF forms and sign-out after each Tor Browser major release.
- **curl**: `curl --socks5-hostname 127.0.0.1:9050 -H "Authorization: Bearer $TOKEN" http://<addr>.onion/api/...` (never `--socks5` without `-hostname`: the name would be resolved locally). Allow 30 to 60 seconds for the first request.
- **Go client**: `proxy.SOCKS5("tcp", "127.0.0.1:9050", nil, proxy.Direct)` from `golang.org/x/net/proxy`, then use its `DialContext` in an `http.Transport`; the host name goes to the proxy unresolved.
- API tokens travel in the `Authorization` header only, never in the URL.

## I2P (optional, second choice)

Differences that matter, from the research doc: no secure context for `.i2p` in ordinary browsers (so `cookie-secure=true` is likely to break sign-in and has to be tested, **unverified**), no uniform browser fingerprint, weaker DoS controls.

i2pd `tunnels.conf` (options named in the i2pd docs, syntax **unverified**):

```ini
[circula]
type = http
host = 127.0.0.1
port = 8080
keys = circula.dat
```

- The `*.b32.i2p` address is derived from `circula.dat`; back that file up like the tor keys.
- Users browse through their router's HTTP proxy (commonly 127.0.0.1:4444, **unverified**).
- Java I2P's HTTP server tunnel adds `X-I2P-DestHash`, `X-I2P-DestB32`, `X-I2P-DestB64`; i2pd's behaviour is **unverified**. Circula does not read them yet (backlog B3); do not put anything in front that forwards a client-supplied `X-I2P-*` header.

## Privacy checklist

- `public-base-url` is the onion (or b32) URL only; no clearnet URL anywhere in templates, invitations or notes.
- No analytics or external assets (current templates load only `/static/style.css` and `/static/app.js`; CSP allows `'self'` only).
- Short log retention on the host; Circula logs no client address in onion mode (peer is loopback).
- Backups go to storage reached over Tor, or accept that the host is then identifiable by where it uploads.
- Keep the clock correct (tor needs it; do not expose the host in other ways).
