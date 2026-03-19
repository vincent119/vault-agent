.PHONY: tidy lint test bench cover fmt vet build run clean

APP_NAME = vault-agent
BUILD_DIR = bin

tidy:
	go mod tidy


lint:
	golangci-lint run ./...

test:
	go test -race -count=1 ./...

bench:
	go test -run=NONE -bench=. -benchmem ./...

cover:
	go test -cover ./...

cover-html:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

fmt:
	go fmt ./...

vet:
	go vet ./...

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) cmd/$(APP_NAME)/main.go

run:
	go run ./cmd/$(APP_NAME)/

clean:
	rm -rf $(BUILD_DIR) coverage.out
