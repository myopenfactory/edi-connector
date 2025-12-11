ARCHS := amd64 arm64
SYSTEMS := linux windows
VERSION := $(shell git describe --always --tags)

build: $(foreach sys, $(SYSTEMS), $(foreach arch, $(ARCHS),build-$(sys)-$(arch)))
dist: $(foreach sys, $(SYSTEMS), $(foreach arch, $(ARCHS),dist-$(sys)-$(arch)))

define BUILD_rule
build-$(1)-$(2):
	CGO_ENABLED=0 GOOS=$(1) GOARCH=$(2) go build -o dist/edi-connector_$(1)_$(2)/ -ldflags="-X github.com/myopenfactory/edi-connector/version.Version=$(VERSION)"

dist-$(1)-$(2): third_party build-$(1)-$(2)
	rm -f dist/edi-connector_$(1)_$(2).zip dist/edi-connector_$(1)_$(2).tar.gz
	cp -R dist/THIRD_PARTY dist/edi-connector_$(1)_$(2)/
	cp LICENSE dist/edi-connector_$(1)_$(2)/
ifeq ($(1), windows)
	cd dist && zip -q -r edi-connector_$(1)_$(2).zip edi-connector_$(1)_$(2)/
else
	cd dist && tar cfz edi-connector_$(1)_$(2).tar.gz edi-connector_$(1)_$(2)/
endif
endef

$(foreach sys, $(SYSTEMS), $(foreach arch, $(ARCHS),$(eval $(call BUILD_rule,$(sys),$(arch)))))

third_party:
	rm -rf dist/THIRD_PARTY || true
	mkdir -p dist
	go-licenses save . --save_path="dist/THIRD_PARTY/"

clean:
	rm -rf dist
