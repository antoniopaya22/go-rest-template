# syntax=docker/dockerfile:1

FROM golang:1.27@sha256:fe517acaa580cc7915457080f850c37e618b3c1e750f98b58fc6429b605147a1 AS build

WORKDIR /src

# Dependencies are downloaded in their own layer so source changes do not invalidate it.
COPY go.mod go.sum ./
RUN go mod download

COPY . ./

# The SQLite driver is pure Go, so the binary can be fully static.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab

WORKDIR /app

ENV APP_ENV=production
ENV HTTP_HOST=0.0.0.0
ENV HTTP_PORT=8080
ENV DATABASE_DRIVER=postgres
ENV AUTO_MIGRATE=true
ENV OPENAPI_SPEC_PATH=/app/api/openapi.yaml

COPY --from=build /out/server /app/server
COPY --from=build /src/api /app/api
COPY --from=build /src/migrations /app/migrations

EXPOSE 8080

ENTRYPOINT ["/app/server"]
