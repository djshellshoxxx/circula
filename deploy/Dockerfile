# syntax=docker/dockerfile:1
# Circula container image: static binary on distroless, non-root, no shell.
# Build from the repository root:  docker build -f deploy/Dockerfile --build-arg VERSION=v1.2.3 -t circula:v1.2.3 .
# OWNER: pin both base images by digest before release (tags are mutable):
#   docker buildx imagetools inspect golang:1.26.10
#   docker buildx imagetools inspect gcr.io/distroless/static-debian12:nonroot
FROM golang:1.26.10 AS build
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown
ENV CGO_ENABLED=0 GOFLAGS=-buildvcs=false GOTOOLCHAIN=local
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
      -o /out/circula ./cmd/circula
# distroless has no shell, so prepare the data dir (uid 65532, mode 0700) here and copy it in.
RUN mkdir -p /out/data && chmod 0700 /out/data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/circula /usr/local/bin/circula
COPY --from=build --chown=65532:65532 /out/data /data
USER 65532:65532
ENV CIRCULA_LISTEN=0.0.0.0:8080 \
    CIRCULA_DATABASE=/data/circula.db \
    CIRCULA_BACKUP_DIR=/data/backups \
    CIRCULA_STATE_DIR=/data
VOLUME ["/data"]
EXPOSE 8080
# No HEALTHCHECK: the image has no shell or curl. Probe GET /healthz and /readyz from the orchestrator or proxy.
ENTRYPOINT ["/usr/local/bin/circula"]
CMD ["serve"]
