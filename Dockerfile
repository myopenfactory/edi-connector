FROM alpine:latest
RUN apk add --no-cache tzdata ca-certificates
COPY edi-connector /edi-connector

LABEL org.opencontainers.image.source="https://github.com/myopenfactory/edi-connector"

ENTRYPOINT ["/edi-connector"]
