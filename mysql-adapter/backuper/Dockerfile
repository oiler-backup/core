FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY . .

RUN go build -o backup-app .

FROM mysql:8.0

COPY --from=builder /app/backup-app /usr/local/bin/backup-app

ENTRYPOINT ["backup-app"]