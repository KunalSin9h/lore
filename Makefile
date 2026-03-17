BINARY    := rememberit
BUILD_DIR := bin

.PHONY: build install test clean

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/rememberit

install:
	go install ./cmd/rememberit

test:
	go test ./...

clean:
	rm -rf $(BUILD_DIR)
