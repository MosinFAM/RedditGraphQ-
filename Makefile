.PHONY: build
build:
	go mod vendor
	go build -mod=vendor -o bin/redditclone ./cmd/redditclone

.PHONY: test
TEST_EXCLUDE = 'github.com/MosinFAM/graphql-posts/(cmd/redditclone|internal/db|internal/models|internal/graph/(generated|model)\.go|internal/storage/(mock_storage|postgres)\.go)'
TEST_PACKAGES = $(shell go list ./... | grep -vE $(TEST_EXCLUDE))
test:
	go test $(TEST_PACKAGES) -coverprofile=coverage.out > /dev/null && \
    grep -vE $(TEST_EXCLUDE) coverage.out > coverage.filtered && \
    mv coverage.filtered coverage.out && \
    go tool cover -html=coverage.out -o cover.html && \
	echo "\nCoverage report:" && \
    go tool cover -func=coverage.out | grep -vE $(TEST_EXCLUDE)

.PHONY: lint
lint:
	go mod vendor
	golangci-lint run -c .golangci.yml -v --modules-download-mode=vendor ./...

.PHONY: clean
clean:
	rm -rf bin/* vendor/*
