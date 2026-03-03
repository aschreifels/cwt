VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -s -w \
           -X github.com/aschreifels/cwt/cmd.version=$(VERSION) \
           -X github.com/aschreifels/cwt/cmd.commit=$(COMMIT) \
           -X github.com/aschreifels/cwt/cmd.date=$(DATE)

.PHONY: build install clean test lint

build:
	go build -ldflags "$(LDFLAGS)" -o cwt .

install: build
	cp cwt $(GOPATH)/bin/cwt 2>/dev/null || cp cwt /usr/local/bin/cwt

clean:
	rm -f cwt
	go clean

test:
	go test ./... -v

lint:
	go vet ./...
