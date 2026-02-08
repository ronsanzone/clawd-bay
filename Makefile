.PHONY: build test lint verify install clean

build:
	go build -o cb main.go

test:
	go test ./...

lint:
	golangci-lint run

verify: test lint
	@echo "All checks passed"

install: build
	cp cb ~/bin/cb

clean:
	rm -f cb
