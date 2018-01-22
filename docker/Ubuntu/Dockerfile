FROM ubuntu
RUN apt-get update && apt-get install -y ca-certificates && /usr/sbin/update-ca-certificates && apt-get clean all
COPY client_linux_amd64 /app/client
COPY ca.pem /app/
ENV CLIENT_CACERT /app/ca.pem
WORKDIR /app/
CMD ["/app/client"]
