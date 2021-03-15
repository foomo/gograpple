pack-data:
	go-bindata -o bindata/bindata.go -pkg bindata the-hook/...

build: pack-data
	go build -o bin/gograpple cmd/main.go

install: pack-data
	go build -o /usr/local/bin/gograpple cmd/main.go

test:
	go test ./...