#!/bin/bash

GO111MODULE=on go install google.golang.org/protobuf/cmd/protoc-gen-go
GO111MODULE=on go install github.com/twitchtv/twirp/protoc-gen-twirp
protoc --proto_path=./api --go_out=./api --twirp_out=./api --go_opt=paths=source_relative --twirp_opt=paths=source_relative api.proto
