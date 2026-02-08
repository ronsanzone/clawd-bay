.PHONY: build test test-integration lint verify install clean run debug

build:
	go build -o cb main.go

run: build
	./cb

debug: build
	./cb --debug

test:
	go test ./...

test-integration:
	go test -tags=integration ./... -v

lint:
	golangci-lint run

verify: test lint
	@echo "All checks passed"

install: build
	cp cb ~/bin/cb

clean:
	rm -f cb
