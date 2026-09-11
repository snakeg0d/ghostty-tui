.PHONY: build test install clean

PREFIX ?= $(HOME)/.local

build:
	go build -trimpath -o bin/ghostty-tui ./cmd/ghostty-tui

test:
	go test ./...

install: build
	install -d "$(PREFIX)/bin" "$(PREFIX)/share/ghostty-studio"
	install -m 755 bin/ghostty-tui "$(PREFIX)/bin/ghostty-tui"
	cp -R assets "$(PREFIX)/share/ghostty-studio/"

clean:
	rm -f bin/ghostty-tui
