FROM golang:1.24 AS builder

COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o main .

ENTRYPOINT [ "./main" ]