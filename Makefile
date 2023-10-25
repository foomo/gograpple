build:
	go build -o bin/gograpple main.go

install:
	go build -o /usr/local/bin/gograpple main.go

test:
	go test ./...