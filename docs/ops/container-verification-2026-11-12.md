# Container verification, 2026-11-12 (matrix O2)

Scope: `deploy/Dockerfile` and `deploy/compose.yaml` at commit bffcede, built and run for the first time.
Result: image builds, runs non-root on a read-only root filesystem with all capabilities dropped, serves
`/healthz` and `/readyz`, and keeps data across container restarts. No defects found; no deploy/ file changed.

## Environment and deviations (sandbox only, not product defects)

- Docker Engine 29.9.2, Compose v5.6.1, root, no systemd. Daemon started with
  `dockerd --iptables=false --ip6tables=false --bridge=none -s vfs`. First start after a cold boot needed a retry
  (containerd start timeout); the second start worked.
- `--bridge=none` means `docker build`/`run` need `--network=host` for outbound access. `docker compose up` still created
  its network and a container IP on `docker0` (172.17.0.2) was reachable from the host, but the sandbox warned
  "IPv4 forwarding is disabled", so port publishing was not tested end to end (only the 127.0.0.1:8080 loopback hit,
  which worked through the userland path).
- The sandbox egress proxy re-signs TLS, so `go mod download` inside the build failed with
  "x509: certificate signed by unknown authority". The real Dockerfile therefore was NOT built unmodified. The build
  used a scratch copy that differs by exactly one added line in the build stage, copying the proxy CA bundle
  (`COPY --from=ca ca-bundle.crt /etc/ssl/certs/ca-certificates.crt` before `WORKDIR /src`), with
  `--build-context ca=/root/.ccr`. The final stage and everything shipped in the image are identical to deploy/Dockerfile.
  The unmodified Dockerfile is expected to build on a normal network but that exact invocation was not run.
- Not tested: image pinned by digest (the file says owner must pin), buildx multi-arch, TLS, a real reverse proxy.

## Commands and results

```
docker build --network=host --build-context ca=/root/.ccr -f <scratch Dockerfile.ca> -t circula:test .
  -> success (~83 s cold, base images pulled); image 19.4 MB on disk
docker image inspect circula:test   -> User 65532:65532, Cmd [serve], Healthcheck <nil>

D="--network=host --read-only --tmpfs /tmp:rw,noexec,nosuid,size=16m --cap-drop ALL \
   --security-opt no-new-privileges:true -v ctest-data:/data \
   -e CIRCULA_PUBLIC_BASE_URL=https://circula.example -e CIRCULA_LISTEN=127.0.0.1:18080"
docker run --rm $D circula:test migrate     -> "pending migrations: v0 to v16 / migrated to v16 / no networks to verify"
docker run -d --name c1 $D circula:test serve
  log: "circula serving ... schema_version":16, startup backup written to /data/backups ("ledger":"ok")
curl -i localhost:18080/healthz              -> 200 OK
curl -i localhost:18080/readyz               -> 200 {"status":"ready"}
docker inspect c1                            -> state=running user=65532:65532 ro=true
```

Data volume persistence and CLI sharing:

```
docker run --rm -i $D circula:test admin create-operator --login ops1   (while c1 running)
  -> "/data/circula.db is in use by another circula process ... stop it first"   (lock works as intended)
docker restart c1  -> /readyz ready again
docker stop c1; echo pw | docker run --rm -i $D circula:test admin create-operator --login ops1
  -> created person ... (login ops1)
docker run -d --name c2 $D circula:test serve  -> /readyz ready; docker stop c2
docker run --rm $D circula:test verify-ledger  -> "no networks to verify" (exit 0)
create-operator ops1 again from a new container -> "conflict: login unavailable"   (row survived restarts)
```

Volume contents as seen on the host: dir mode 755 owner 65532; `circula.db`, `circula.db.lock`, `circula-web.key`
all 0600 owned by 65532; `backups/` 0700. The image's 0700 /data is not kept on a fresh named volume, but all files
inside are 0600, so this is acceptable.

Compose (`CIRCULA_PUBLIC_BASE_URL=https://circula.example`):

```
docker compose -f deploy/compose.yaml config -q   -> ok
docker compose -f deploy/compose.yaml up -d --build    (not used; see build deviation)
docker tag circula:test circula:local
docker compose -f deploy/compose.yaml up -d --no-build -> Up, ports 127.0.0.1:8080->8080/tcp
  log: "migrations applied from 0 to 16", then "circula serving ... listen 0.0.0.0:8080"
curl 172.17.0.2:8080/readyz -> {"status":"ready"}; curl 127.0.0.1:8080/healthz -> 200
docker inspect -> ro=true caps=[ALL] dropped user=65532:65532
docker compose down -> ok
```

## Observations (no change made)

1. On a brand-new empty database `serve` applies migrations itself even with `migrate-mode=explicit`
   (log "migrations applied from 0 to 16"). The compose.yaml/Dockerfile comment says to run `migrate` once before first
   start; that is safe but not required for a first install. Pending migrations on an existing database were not
   exercised here (upgrade.md covers it). Owner may want the comment reworded; not changed.
2. No HEALTHCHECK by design (distroless, no shell/curl); documented in the Dockerfile. Compose has no healthcheck either.
   Orchestrator must probe `/healthz`/`/readyz`.
3. Compose never sets a setup secret, backup key or metrics token; the container logs warnings for all of them. Expected
   for the example file; docs/ops/hardening.md covers them.
4. `CIRCULA_TRUSTED_PROXIES` default `172.16.0.0/12` suits the compose bridge only.

Tooling cleanup: containers, volumes and the compose network were removed; dockerd was stopped at the end.
