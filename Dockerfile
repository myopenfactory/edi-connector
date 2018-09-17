FROM golang:1.11.0-alpine AS build

ARG VERSION

RUN apk add --update \
		gcc \
		musl-dev \
		git

WORKDIR /client
COPY . /client/
RUN go build -ldflags "-X github.com/myopenfactory/client/cmd.version=$VERSION"

FROM alpine:latest
RUN apk add --no-cache tzdata ca-certificates
COPY myOpenFactoryCA.crt /usr/local/share/ca-certificates/extra/myOpenFactoryCA.crt
RUN update-ca-certificates

WORKDIR /app/
COPY --from=build /client/client /app/client
CMD ["/app/client"]