build:
	go build -o bin/gograpple cmd/gograpple/main.go

install:
	go build -o /usr/local/bin/gograpple cmd/gograpple/main.go

test:
	go test ./...