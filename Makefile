VERSION=$(shell sh version.sh)

.PHONY: all test cover clean

all: fex fex.1

test:
	go test ./...

cover:
	go test -coverprofile=cover.out ./...

fex: cmd/fex/fex.go cmd/fex/fex_test.go VERSION
	go build -ldflags "-X main.version=$(VERSION)" ./cmd/fex

fex.1: README.adoc
	asciidoctor -b manpage -o $@ $<

clean:
	$(RM) fex fex.1
	go clean
