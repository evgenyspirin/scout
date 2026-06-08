# ---- build stage ----
FROM golang:1.25-alpine AS build
WORKDIR /src

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Build static binaries (pure-Go SQLite driver, so CGO can stay disabled)
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/scout ./cmd/scout \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/seeder-images ./cmd/seeder-images

# ---- runtime stage ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 scout
WORKDIR /app
COPY --from=build /out/scout /app/scout
COPY --from=build /out/seeder-images /app/seeder-images
USER scout
EXPOSE 8080
ENTRYPOINT ["/app/scout"]
