# Makefile for cockpit
GOWORK := off

.PHONY: build install uninstall clean

build:
	GOWORK=$(GOWORK) go build -o cockpit ./cmd/cockpit

install: build
	mkdir -p $(HOME)/.local/bin
	cp -f ./cockpit $(HOME)/.local/bin/cockpit
	chmod +x $(HOME)/.local/bin/cockpit
	@echo "cockpit installed to $(HOME)/.local/bin/cockpit"

uninstall:
	rm -f $(HOME)/.local/bin/cockpit
	@echo "cockpit uninstalled"

clean:
	rm -f ./cockpit