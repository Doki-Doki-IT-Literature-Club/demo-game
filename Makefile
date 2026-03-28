lint:
	go fmt ./...

client:
	go run cmd/client/main.go ${addr}

server:
	go run cmd/server/main.go ${port}

map_editor:
	go run cmd/mapeditor/main.go

build:
	mkdir bin
	go build -o bin/client cmd/client/main.go
	go build -o bin/server cmd/server/main.go
