.PHONY: build test test-integration lint verify install install-local clean run debug smoke

MODULE := github.com/rsanzone/clawdbay

build:
	go build -o cb main.go

run: build
	./cb

debug: build
	./cb --debug

smoke: build
	./cb --help

test:
	go test ./...

test-integration:
	go test -tags=integration ./... -v

lint:
	golangci-lint run

verify: test lint
	@echo "All checks passed"

install:
	go install $(MODULE)@latest

install-local: build
	cp cb ~/bin/cb

clean:
	rm -f cb
