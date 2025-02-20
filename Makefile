.PHONY: build
build:
	go mod vendor
	go build -mod=vendor -o bin/redditclone ./cmd/redditclone

.PHONY: test
test:
	go clean -testcache
	go test -v -coverpkg=./... -coverprofile=coverage.out -covermode=count ./internal/graph && \
	go tool cover -func=coverage.out | grep -v 'generated.go' | awk '{print $$3}' && \
	go tool cover -html=coverage.out -o cover.html


.PHONY: lint
lint:
	go mod vendor
	golangci-lint run -c .golangci.yml -v --modules-download-mode=vendor ./...

.PHONY: clean
clean:
	rm -rf bin/* vendor/*
