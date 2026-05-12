.PHONY: all build test vet manifest-checksum

all: build

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

manifest-checksum:
	@sha256sum cmd/continuum-plugin-audiobooksdb/manifest.json | cut -d' ' -f1
