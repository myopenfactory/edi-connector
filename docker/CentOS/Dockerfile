FROM centos
RUN yum install ca-certificates && /bin/update-ca-trust && yum clean all
COPY client_linux_amd64 /app/client
COPY ca.pem /app/
ENV CLIENT_CACERT /app/ca.pem
WORKDIR /app/
CMD ["/app/client"]
