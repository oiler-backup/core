FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY . .

RUN go build -o backup-restore-app .

FROM mysql:8.0

COPY --from=builder /app/backup-restore-app /usr/local/bin/backup-restore-app

ENTRYPOINT ["backup-restore-app"]