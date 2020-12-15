BUILD_COMMIT := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -Iseconds)
ifeq ($(OS),Windows_NT)
	BUILD_OUTPUT_EXTENSION=.exe
	COMPOSE_FILE=docker-compose.windows.yml
else
	COMPOSE_FILE=docker-compose.yml
endif

build:
	CGO_ENABLED=0 go build -o myof-client${BUILD_OUTPUT_EXTENSION} -ldflags "-X github.com/myopenfactory/client/pkg/version.Date=${BUILD_DATE} -X github.com/myopenfactory/client/pkg/version.Commit=${BUILD_COMMIT}"
	CGO_ENABLED=0 go build -o ./test/e2e-test${BUILD_OUTPUT_EXTENSION} ./test

generate:
	go generate ./...

protogen:
	@./protogen.sh

.PHONY: test
test: build
	cd test && docker-compose -f ${COMPOSE_FILE} down -v
	cd test && docker-compose -f ${COMPOSE_FILE} up --build --force-recreate --abort-on-container-exit --renew-anon-volumes --exit-code-from test
