FROM golang:1.11.1-alpine AS build

ARG VERSION
ENV GOPROXY=https://modules.myopenfactory.io

WORKDIR /client
COPY . /client/
RUN CGO_ENABLED=0 go build -ldflags "-X github.com/myopenfactory/client/cmd.version=$VERSION"

FROM alpine:latest
RUN apk add --no-cache tzdata ca-certificates
COPY myOpenFactoryCA.crt /usr/local/share/ca-certificates/extra/myOpenFactoryCA.crt
RUN update-ca-certificates

WORKDIR /app/
COPY --from=build /client/client /app/client
CMD ["/app/client"]