FROM gcr.io/distroless/static-debian12:nonroot AS amd64
COPY dist/goreleaser/edi-connector_linux_amd64_v1/edi-connector /edi-connector

FROM gcr.io/distroless/static-debian12:nonroot AS arm64
COPY dist/goreleaser/edi-connector_linux_arm64_v8.0/edi-connector /edi-connector

FROM $TARGETARCH
LABEL org.opencontainers.image.source="https://github.com/myopenfactory/edi-connector"

VOLUME /data
ENTRYPOINT ["/edi-connector", "--config", "/data/config.json"]
