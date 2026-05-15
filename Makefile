BINARY := continuum-plugin-local-audiobooks
GO ?= go

.PHONY: all build test vet manifest-checksum clean

all: build

build:
	$(GO) build -o $(BINARY) ./cmd/continuum-plugin-local-audiobooks

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

manifest-checksum:
	@sha256sum cmd/continuum-plugin-local-audiobooks/manifest.json | cut -d' ' -f1

clean:
	rm -f $(BINARY)
