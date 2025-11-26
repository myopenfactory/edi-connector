build: third_party
	goreleaser build --snapshot --clean
	
third_party:
	rm -rf dist/THIRD_PARTY || true
	mkdir -p dist
	go-licenses save . --save_path="dist/THIRD_PARTY/"
