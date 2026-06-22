# Multi-stage build for cmd/iscc-monitor: a CGO_ENABLED=0 static Go binary into a
# minimal, non-root final image with CA roots present (ADR-0013, M-Deploy).
#
# Build:  docker build --build-arg VERSION="$(git rev-parse --short HEAD)" -t iscc-monitor .
# Locally a caller may pass a literal, e.g. --build-arg VERSION=dev.
#
# VERSION is a REQUIRED, non-empty build-arg: it is injected at link time via
# `-ldflags -X internal/version.Version`, which GET /version reports. An empty -X
# value OVERRIDES the in-code "dev" default (it is not a no-op), so an empty stamp
# would silently ship a blank /version and defeat the build provenance. The build
# RUN therefore fails fast on an empty VERSION rather than producing such an image.
# It is deliberately NOT defaulted (ARG VERSION=dev) — defaulting would mask the
# very trap the fail-fast guard exists to catch; the caller must pass a value.

# --- Stage 1: build the static binary --------------------------------------
# Pin the concrete toolchain tag for reproducibility; matches go.mod's
# `go 1.26.1` directive and mise.toml's `go = "1.26.4"`.
FROM golang:1.26.4 AS build
WORKDIR /src

# Cache module downloads independently of the source tree.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION
RUN [ -n "$VERSION" ] || { echo 'VERSION build-arg is empty — refusing to ship an empty /version stamp'; exit 1; } && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath \
      -ldflags "-s -w -X github.com/iscc/iscc-monitor/internal/version.Version=${VERSION}" \
      -o /iscc-monitor ./cmd/iscc-monitor

# --- Stage 2: minimal non-root final image ---------------------------------
# distroless/static-debian12:nonroot is scratch-class for a static binary, ships
# the CA root bundle (needed for runtime did:web + hub HTTPS, ADR-0009), and runs
# as the unprivileged uid 65532 (nonroot) — "non-root" + "CA roots present" in one
# base, no /etc/passwd plumbing required.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /iscc-monitor /iscc-monitor

# Bake the interim realm document so a fresh container has a valid
# ISCC_MONITOR_REALM out of the box. This is the existing testnet pilot fixture;
# the canonical deploy/realm-testnet.txt is a later M-Deploy slice.
COPY internal/registry/testdata/realm.txt /etc/iscc-monitor/realm.txt

# Documents the default ISCC_MONITOR_ADDR port (:9464); publishes nothing by itself.
EXPOSE 9464

ENTRYPOINT ["/iscc-monitor"]
