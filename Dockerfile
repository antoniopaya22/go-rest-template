# syntax=docker/dockerfile:1

# The build stage runs on the build machine and cross-compiles, so multi-arch
# images do not need emulation.
FROM --platform=$BUILDPLATFORM golang:1.27@sha256:fe517acaa580cc7915457080f850c37e618b3c1e750f98b58fc6429b605147a1 AS build

WORKDIR /src

# Dependencies are downloaded in their own layer so source changes do not invalidate it.
COPY go.mod go.sum ./
RUN go mod download

COPY . ./

ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown
ARG TARGETOS
ARG TARGETARCH

# The database drivers are pure Go, so the binaries are fully static.
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath \
      -ldflags="-s -w \
        -X github.com/antoniopaya22/go-rest-template/internal/platform/version.Version=${VERSION} \
        -X github.com/antoniopaya22/go-rest-template/internal/platform/version.Commit=${COMMIT} \
        -X github.com/antoniopaya22/go-rest-template/internal/platform/version.Date=${DATE}" \
      -o /out/ ./cmd/api ./cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab

LABEL org.opencontainers.image.source="https://github.com/antoniopaya22/go-rest-template" \
      org.opencontainers.image.licenses="MIT"

WORKDIR /app

# The OpenAPI document and the migrations are embedded in the binaries.
COPY --from=build /out/api /app/server
COPY --from=build /out/migrate /app/migrate

ENV APP_ENV=production

EXPOSE 8080 9090

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 CMD ["/app/server", "-healthcheck"]

ENTRYPOINT ["/app/server"]
