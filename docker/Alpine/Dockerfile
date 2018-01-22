FROM alpine
RUN apk add --no-cache ca-certificates && update-ca-certificates
COPY client_linux_amd64 /app/client
COPY ca.pem /app/
ENV CLIENT_CACERT /app/ca.pem
WORKDIR /app/
CMD ["/app/client"]
