BINARY   := aikits
MODULE   := github.com/qunv/aikits
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  := -ldflags "-X '$(MODULE)/internal/command.Version=$(VERSION)'"

# Install destination: prefer GOBIN, fall back to GOPATH/bin, then /usr/local/bin.
GOBIN    := $(or $(shell go env GOBIN),$(shell go env GOPATH)/bin)
DESTDIR  ?= $(GOBIN)

.PHONY: build run test lint clean tidy install uninstall dev

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/$(BINARY)

dev:
	WEBKIT_DISABLE_DMABUF_RENDERER=1 wails dev -tags webkit2_41

install: build
	@mkdir -p $(DESTDIR)
	cp bin/$(BINARY) $(DESTDIR)/$(BINARY)
	@echo "✅  $(BINARY) installed to $(DESTDIR)/$(BINARY)"
	@echo "   Make sure $(DESTDIR) is in your PATH."

uninstall:
	rm -f $(DESTDIR)/$(BINARY)
	@echo "🗑️  $(BINARY) removed from $(DESTDIR)"

run:
	go run $(LDFLAGS) ./cmd/$(BINARY) $(ARGS)

test:
	go test ./... -race -cover

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

clean:
	rm -rf bin/
