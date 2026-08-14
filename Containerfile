# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/dungeogo ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/dungeogo /dungeogo

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/dungeogo"]
