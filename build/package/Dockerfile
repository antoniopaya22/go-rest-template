FROM golang:1.26 AS build

WORKDIR /src

COPY go.mod ./
COPY . ./

RUN go mod tidy
RUN go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/api

FROM gcr.io/distroless/base-debian12:nonroot

WORKDIR /app

ENV APP_ENV=production
ENV HTTP_HOST=0.0.0.0
ENV HTTP_PORT=8080
ENV DATABASE_DRIVER=postgres
ENV SQLITE_PATH=/app/data/app.db
ENV AUTO_MIGRATE=true
ENV OPENAPI_SPEC_PATH=/app/api/openapi.yaml

COPY --from=build /out/server /app/server
COPY --from=build /src/api /app/api
COPY --from=build /src/migrations /app/migrations
COPY --from=build /src/data /app/data

EXPOSE 8080

ENTRYPOINT ["/app/server"]