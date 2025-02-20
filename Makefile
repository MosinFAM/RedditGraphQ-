.PHONY: build
build:
	go mod vendor
	go build -mod=vendor -o bin/redditclone ./cmd/redditclone

.PHONY: test
test:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o cover.html

.PHONY: lint
lint:
	go mod vendor
	golangci-lint run -c .golangci.yml -v --modules-download-mode=vendor ./...

.PHONY: clean
clean:
	rm -rf bin/* vendor/*
