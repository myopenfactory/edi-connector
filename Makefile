build: third_party
	goreleaser build --snapshot --clean
	
third_party:
	rm -rf THIRD_PARTY || true
	go-licenses save . --save_path="THIRD_PARTY/"

.PHONY: installer
installer: third_party build
	export VERSION=$(shell eval jq -r '.version' ./dist/metadata.json); makensis hacks/installer.nsi
