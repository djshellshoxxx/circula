# Hardening: systemd unit and container

Least-privilege recipes for running `circula serve`. Files live in [`deploy/`](../../deploy). Settings names are from
[configuration.md](configuration.md). The process already sets umask `077`, creates its files `0600` and directories
`0700`, reads secrets from files with a permission check, and makes no outbound connections; the sandbox below
enforces the same from outside.

## systemd (bare host)

```sh
install -m 0755 circula /usr/local/bin/circula              # static binary: CGO_ENABLED=0 go build -trimpath ./cmd/circula
install -m 0644 deploy/circula.sysusers /usr/lib/sysusers.d/circula.conf && systemd-sysusers
install -m 0644 deploy/circula.tmpfiles /usr/lib/tmpfiles.d/circula.conf && systemd-tmpfiles --create
install -m 0644 deploy/circula.service /etc/systemd/system/circula.service
systemctl edit circula        # set CIRCULA_PUBLIC_BASE_URL and anything else (drop-in, see below)
sudo -u circula env CIRCULA_DATABASE=/var/lib/circula/circula.db /usr/local/bin/circula migrate
systemctl enable --now circula
```

`migrate-mode` is `explicit` by default, so run `circula migrate` once before the first start and on every upgrade
([upgrade.md](upgrade.md)).

Secrets go in a drop-in, never in the unit, using credentials (files are mode 0400, owned by the service user, and
visible only to the service, which satisfies the "not readable by group or others" check):

```ini
# /etc/systemd/system/circula.service.d/local.conf
[Service]
LoadCredential=setup-secret:/etc/circula/setup-secret
Environment=CIRCULA_SETUP_SECRET_FILE=%d/setup-secret
```

What the sandbox does: runs as an unprivileged user with an empty capability set, `NoNewPrivileges`, read-only
filesystem except `/var/lib/circula` (`ProtectSystem=strict`, `ReadWritePaths`), no home directories, private `/tmp`
and `/dev`, kernel tunables/modules/logs/cgroups/clock protected, a `@system-service` syscall allow-list minus
`@privileged @resources`, native syscall ABI only, no W+X memory (`MemoryDenyWriteExecute`; fine for a static Go binary
without JIT), only `AF_UNIX/INET/INET6` sockets, and loopback-only IP access (`IPAddressDeny=any`,
`IPAddressAllow=localhost`). Binding below port 1024 is not allowed: listen on a high port behind the proxy. If the
listener is on a LAN address, delete the two `IPAddress*` lines and use firewall rules.

Moving the backup directory: set `CIRCULA_BACKUP_DIR` and add the path to `ReadWritePaths=` in the drop-in.

### Verification status

- `systemd-analyze verify deploy/circula.service` (systemd 255): passes; the only message is that
  `/usr/local/bin/circula` does not exist on the build host.
- `systemd-analyze security --offline=yes deploy/circula.service`: overall exposure 1.1 OK.
- `scripts/smoke-deploy.sh` runs the binary with the unit's `ExecStart` and `Environment=` settings (it fails if the unit
  drifts) and checks `/readyz`, `/healthz`, the database mode and a clean SIGTERM stop. It does **not** apply the
  sandbox. Starting the real unit (seccomp filter, `MemoryDenyWriteExecute`, `IPAddressDeny`) was **not** tested in
  the build environment (no running systemd). Do this once on the target host and watch for `status=31/SYS` (seccomp
  kill):

  ```sh
  systemctl start circula && systemctl status circula && curl -fsS http://127.0.0.1:8080/readyz
  systemd-analyze security circula
  ```

  If a syscall is blocked on your kernel/libc, `journalctl -u circula` and `ausearch -m SECCOMP` show which; relax the
  filter minimally rather than dropping it.

## Container

```sh
docker build -f deploy/Dockerfile --build-arg VERSION=v1.2.3 --build-arg COMMIT=$(git rev-parse --short=12 HEAD) \
  --build-arg DATE=$(git log -1 --format=%cI) -t circula:v1.2.3 .
docker volume create circula-data
docker run --rm -v circula-data:/data circula:v1.2.3 migrate
docker run -d --name circula --read-only --tmpfs /tmp:rw,noexec,nosuid,size=16m \
  --cap-drop ALL --security-opt no-new-privileges --pids-limit 256 --memory 512m \
  -v circula-data:/data -p 127.0.0.1:8080:8080 \
  -e CIRCULA_PUBLIC_BASE_URL=https://circula.example.org -e CIRCULA_TRUSTED_PROXIES=172.16.0.0/12 \
  circula:v1.2.3 serve
```

[`deploy/compose.yaml`](../../deploy/compose.yaml) is the same as a compose file. The image is a static binary on
`gcr.io/distroless/static-debian12:nonroot` (no shell, no package manager), user `65532`, one writable volume `/data`
(database, backups, `circula-web.key`). Pass secrets as read-only mounted files with `CIRCULA_*_FILE`, mode `0600`
and owned by uid 65532 (a secret file readable by group or others is refused). There is no `HEALTHCHECK` because the
image has no tool to run one; probe `/healthz` and `/readyz` from the orchestrator or proxy.

Inside a container `CIRCULA_LISTEN` is `0.0.0.0:8080`; publish the port to loopback or an internal network only, and
keep `/metrics` unreachable from outside (see [monitoring.md](monitoring.md)).

Status: `deploy/Dockerfile` and the commands above were **not built or run** in the authoring environment (no Docker
daemon). The binary they package is built and smoke-tested by `scripts/smoke-deploy.sh`. Before release, pin both base
images by digest (see the comment in the Dockerfile) and run `docker run` as above once.

## Reproducible builds

`scripts/check-reproducible.sh` (CI job `reproducible`) builds the binary twice with the release flags
(`CGO_ENABLED=0`, `-trimpath`, `-buildvcs=false`, version stamps from the commit and its commit time, never the wall
clock), the second time from a copy of the tree in another directory with a fresh build cache, and compares SHA-256.
Locally the two builds were identical. Reproducibility holds for the same Go toolchain, OS and architecture; the
`go.mod` toolchain line fixes the version.

## Not done

Landlock confinement, Go-level seccomp, and PIE builds are not provided; the systemd filter above is the supported
sandbox.
