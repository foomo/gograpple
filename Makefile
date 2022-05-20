build:
	go build -o bin/gograpple cmd/main.go

install:
	go build -o /usr/local/bin/gograpple cmd/main.go

test:
	go test ./...