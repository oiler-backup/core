FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY . .

RUN go build -o backup-app .

FROM alpine:latest

RUN apk add mongodb-tools

COPY --from=builder /app/backup-app /usr/local/bin/backup-app

ENTRYPOINT ["backup-app"]