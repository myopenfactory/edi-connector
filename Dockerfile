FROM golang:1.11rc1-alpine3.8 AS builder

ARG BUILD

RUN apk add --update \
		gcc \
		musl-dev

WORKDIR /client
COPY go.mod go.sum /client/
ENV GOPROXY="https://modules.myopenfactory.io"
COPY . /client
RUN cd /client/cmd && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$BUILD" -o /build/client_linux_amd64
RUN cd /client/cmd && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=$BUILD" -o /build/client_windows_amd64.exe

FROM alpine:latest AS alpine
RUN apk --no-cache add tzdata zip ca-certificates
WORKDIR /usr/share/zoneinfo
# -0 means no compression.  Needed because go's
# tz loader doesn't handle compressed data.
RUN zip -r -0 /zoneinfo.zip .

FROM scratch
ENV ZONEINFO /zoneinfo.zip
COPY --from=alpine /zoneinfo.zip /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/client_linux_amd64 /app/client
WORKDIR /app/
CMD ["/app/client"]
