generate:
	go generate ./...

protogen:
	@./protogen.sh

third_party:
	rm -rf THIRD_PARTY || true
	go install github.com/google/go-licenses@latest
	go-licenses save . --save_path="THIRD_PARTY/"

.PHONY: installer
installer: third_party
	GOOS=windows GOARCH=amd64 goreleaser build --snapshot --clean --single-target
	rm hacks/myof-client_installer.exe || true
	export VERSION=$(shell eval jq -r '.version' ./dist/metadata.json); makensis hacks/installer.nsi

