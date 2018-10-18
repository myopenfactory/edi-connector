FROM golang:1.11.0-alpine

RUN apk add --update \
		gcc \
		musl-dev \
		git

WORKDIR /app
COPY . /app/
RUN go build -o test.bin ./test
CMD ["/app/test.bin"]