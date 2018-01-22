FROM microsoft/nanoserver
COPY client_windows_amd64.exe /app/client
COPY ca.pem /app/
ENV CLIENT_CACERT /app/ca.pem
WORKDIR /app/
CMD ["/app/client"]
