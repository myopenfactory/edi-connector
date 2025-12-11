FROM gcr.io/distroless/static-debian12:nonroot
LABEL org.opencontainers.image.source="https://github.com/myopenfactory/edi-connector"

COPY dist/edi-connector_linux_${TARGETARCH}/edi-connector /edi-connector

VOLUME /data
ENTRYPOINT ["/edi-connector", "--config", "/data/config.json"]
