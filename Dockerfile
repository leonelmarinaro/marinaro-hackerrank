# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/api ./cmd/api

FROM alpine:3.20
WORKDIR /app

RUN addgroup -S app && adduser -S app -G app

COPY --from=builder /out/api /app/api
COPY testdata/products.json /app/testdata/products.json

ENV GIN_MODE=release
ENV PORT=8080
ENV PRODUCTS_FILE=/app/testdata/products.json

USER app
EXPOSE 8080

ENTRYPOINT ["/app/api"]
