test:
	go test ./...
.PHONY: test

lint:
	go vet ./...
.PHONY: lint

run-motivation:
	go run ./_examples/motivation
.PHONY: run-motivation