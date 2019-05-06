BUILD_COMMIT := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -Iseconds)

build:
	CGO_ENABLED=0 go build -o myof-client -ldflags "-X github.com/myopenfactory/client/pkg/version.Date=${BUILD_DATE} -X github.com/myopenfactory/client/pkg/version.Commit=${BUILD_COMMIT}"

generate:
	go generate ./...

release:
	docker run --rm --privileged -v ${PWD}:/src/github.com/myopenfactory/client -v /var/run/docker.sock:/var/run/docker.sock -w /src/github.com/myopenfactory/client -e SIGN_KEY=myOpenFactory_Development.pem goreleaser/goreleaser release --skip-publish --rm-dist --skip-validate --debug

.PHONY: test
.ONESHELL:
test: build
	docker build -t client:latest .
	cd internal/testing
	go run main.go
	# docker-compose up --build --force-recreate --abort-on-container-exit --renew-anon-volumes --exit-code-from test --timeout 30