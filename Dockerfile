# ---- build stage ----
FROM golang:1.25-alpine AS build
WORKDIR /app

# Dependencies for build and time zone data
RUN apk add --no-cache git ca-certificates tzdata curl

# Go dependencies
COPY go.mod go.sum ./
RUN go mod download

# App source
COPY . .

# Build static binary for Linux/amd64
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -ldflags="-s -w" -o server ./main.go

# ---- run stage ----
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

# Timezone
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
ENV TZ=Asia/Tokyo

# App binary
COPY --from=build /app/server /app/server

# App listens on :8080 (set in main.go)
EXPOSE 8080
ENV AWS_LWA_PORT=8080

# Run as non-root
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
