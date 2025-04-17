FROM golang:1.24-alpine AS builder

RUN apk add --no-cache postgresql-client git

WORKDIR /app
COPY . .

RUN go build -o backup-restore-app .

FROM alpine:latest

RUN apk add --no-cache postgresql-client

COPY --from=builder /app/backup-restore-app /usr/local/bin/backup-restore-app

ENTRYPOINT ["backup-restore-app"]