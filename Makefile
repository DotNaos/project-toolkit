BINARY=bin/repo-kit

build:
	go build -o $(BINARY) ./cli

test:
	go test ./...
