FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod .
RUN go mod download

COPY . .
RUN go build -o s3-cleaner .

FROM alpine:latest

COPY --from=builder /app/s3-cleaner /usr/local/bin/s3-cleaner

ENTRYPOINT ["s3-cleaner"]