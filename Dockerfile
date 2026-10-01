# syntax=docker/dockerfile:1

# 1. Telegram Mini App
FROM node:22-alpine AS webapp
WORKDIR /webapp
COPY webapp/package.json webapp/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY webapp/ ./
RUN npm run build

# 2. API
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/lewe-api ./cmd/api
# Upload directory owned by the distroless "nonroot" user (65532), so a
# volume mounted there starts out writable.
RUN mkdir -p /out/data/uploads && chown -R 65532:65532 /out/data

# 3. Runtime
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/lewe-api /lewe-api
COPY --from=webapp /webapp/dist /webapp
COPY --from=build --chown=65532:65532 /out/data /data
ENV WEBAPP_DIR=/webapp UPLOAD_DIR=/data/uploads
EXPOSE 8080
ENTRYPOINT ["/lewe-api"]
