# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/lewe-api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/lewe-api /lewe-api
EXPOSE 8080
ENTRYPOINT ["/lewe-api"]
