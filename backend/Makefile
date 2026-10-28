.PHONY: all build run test clean docker-build cli

all: test build

build:
	go build -o bin/server cmd/server/main.go
	go build -o bin/commitlint-cli cmd/cli/main.go

run:
	go run cmd/server/main.go

cli:
	go run cmd/cli/main.go

test:
	go test -v ./...

clean:
	rm -rf bin/ server.exe err.log out.log

docker-build:
	docker build -t commitlint-ai-backend .
