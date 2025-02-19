build: third_party
	goreleaser build --snapshot --clean --single-target
	
third_party:
	rm -rf THIRD_PARTY || true
	go install github.com/google/go-licenses@latest
	go-licenses save . --save_path="THIRD_PARTY/"

.PHONY: yaml_plugin
yaml_plugin:
	cd ~/src/github.com/tobiaskohlbau/nsis-yaml && zig build plugin -Doptimize=ReleaseSmall && cp ~/src/github.com/tobiaskohlbau/nsis-yaml/zig-out/bin/nsYaml.dll ~/src/github.com/myopenfactory/edi-connector/hacks/plugins/

.PHONY: installer
installer: third_party yaml_plugin
	GOOS=windows GOARCH=amd64 goreleaser build --snapshot --clean --single-target
	rm hacks/edi-connector_installer.exe || true
	export VERSION=$(shell eval jq -r '.version' ./dist/metadata.json); makensis hacks/installer.nsi

deploy: installer
	cp hacks/edi-connector_installer.exe /home/tobiaskohlbau/shared/Windows11/
